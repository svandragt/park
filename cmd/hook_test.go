package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/svandragt/park/internal/park"
)

// newTestRepo creates a git repo with the given origin remote and returns its path.
func newTestRepo(t *testing.T, remote string) string {
	t.Helper()
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", "--quiet"},
		{"remote", "add", "origin", remote},
	} {
		c := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	return dir
}

// usageRows returns the usage table as sorted "uid event source" strings.
func usageRows(t *testing.T, store *park.Store) []string {
	t.Helper()
	rows, err := store.DB().Query(`SELECT item_uid, event, source FROM usage`)
	if err != nil {
		t.Fatalf("select usage: %v", err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var uid, event, source string
		if err := rows.Scan(&uid, &event, &source); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, uid+" "+event+" "+source)
	}
	sort.Strings(got)
	return got
}

// usageFor builds the usageRows entries for the given items.
func usageFor(t *testing.T, store *park.Store, event, source string, ids ...int64) []string {
	t.Helper()
	var want []string
	for _, id := range ids {
		it, err := store.Get(id)
		if err != nil {
			t.Fatalf("get %d: %v", id, err)
		}
		want = append(want, it.UID+" "+event+" "+source)
	}
	sort.Strings(want)
	return want
}

func TestHookRun_EmitsSessionContext(t *testing.T) {
	store := newTestStore(t)
	dir := newTestRepo(t, "https://github.com/svandragt/park")
	if _, err := store.Add(park.Item{Name: "Fix the widget", Remote: "https://github.com/svandragt/park"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	var out bytes.Buffer
	if err := hookRun(store, strings.NewReader(`{"cwd":"`+dir+`"}`), &out, true); err != nil {
		t.Fatalf("hookRun: %v", err)
	}

	var got struct {
		SystemMessage      string `json:"systemMessage"`
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
	}
	if got.HookSpecificOutput.HookEventName != "SessionStart" {
		t.Errorf("hookEventName = %q, want SessionStart", got.HookSpecificOutput.HookEventName)
	}
	ctx := got.HookSpecificOutput.AdditionalContext
	if !strings.Contains(ctx, "Fix the widget") {
		t.Errorf("additionalContext missing item name:\n%s", ctx)
	}
	if !strings.Contains(ctx, "do not act on it unprompted") {
		t.Errorf("additionalContext missing the do-not-act clause:\n%s", ctx)
	}
	// The user only ever sees systemMessage; additionalContext goes to the
	// agent silently, which reads as "the hook never fired".
	if !strings.Contains(got.SystemMessage, "Fix the widget") {
		t.Errorf("systemMessage missing item name:\n%s", got.SystemMessage)
	}
}

// Claude Code shows the agent only the first 2 KB of hook output, so each
// item gets one line and the agent searches for the detail.
func TestHookRun_OneLinePerItem(t *testing.T) {
	store := newTestStore(t)
	dir := newTestRepo(t, "https://github.com/svandragt/park")
	id, err := store.Add(park.Item{
		Name:        "Fix the widget",
		Description: "The widget leaks memory",
		Tags:        "BH-1,widget",
		Remote:      "https://github.com/svandragt/park",
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := store.Add(park.Item{Name: "Untagged item", Remote: "https://github.com/svandragt/park"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	ctx := hookContext(t, store, dir, false)

	if want := fmt.Sprintf("#%d  Fix the widget  [BH-1,widget]\n", id); !strings.Contains(ctx, want) {
		t.Errorf("additionalContext missing line %q:\n%s", want, ctx)
	}
	if !strings.Contains(ctx, "  Untagged item\n") || strings.Contains(ctx, "Untagged item  [") {
		t.Errorf("untagged item should have no tags part:\n%s", ctx)
	}
	for _, unwanted := range []string{"The widget leaks memory", "https://github.com/svandragt/park"} {
		if strings.Contains(ctx, unwanted) {
			t.Errorf("additionalContext should not contain %q:\n%s", unwanted, ctx)
		}
	}
	if !strings.Contains(ctx, "park search") {
		t.Errorf("additionalContext should tell the agent to search:\n%s", ctx)
	}
}

// The compact index drops the bytes the preview cannot spare: tags the name
// or an earlier tag already contains, and the "#" before each id.
func TestHookRun_CompactDropsRedundantTags(t *testing.T) {
	store := newTestStore(t)
	dir := newTestRepo(t, "https://github.com/svandragt/park")
	id, err := store.Add(park.Item{
		Name:   "BH-716: fix the widget",
		Tags:   "bh,BH-716,widget,consent,Consent",
		Remote: "https://github.com/svandragt/park",
	})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, err := store.Add(park.Item{Name: "Untagged item", Tags: "untagged", Remote: "https://github.com/svandragt/park"}); err != nil {
		t.Fatalf("add: %v", err)
	}

	ctx := hookContext(t, store, dir, true)

	if want := fmt.Sprintf("\n%d BH-716: fix the widget [consent]\n", id); !strings.Contains(ctx, want) {
		t.Errorf("additionalContext missing line %q:\n%s", want, ctx)
	}
	if !strings.Contains(ctx, " Untagged item\n") {
		t.Errorf("item whose tags all repeat its name should have no tags part:\n%s", ctx)
	}
	for _, want := range []string{"do not act on it unprompted", "park search", "park list --tag"} {
		if !strings.Contains(ctx, want) {
			t.Errorf("compact preamble missing %q:\n%s", want, ctx)
		}
	}
}

func TestRunHook_CompactFlag(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want bool
	}{
		{[]string{"run"}, true},
		{[]string{"run", "--compact=false"}, false},
	} {
		got, err := hookRunCompact(tc.args[1:])
		if err != nil || got != tc.want {
			t.Errorf("hookRunCompact(%v) = %v, %v; want %v", tc.args[1:], got, err, tc.want)
		}
	}
}

func hookContext(t *testing.T, store *park.Store, dir string, compact bool) string {
	t.Helper()
	var out bytes.Buffer
	if err := hookRun(store, strings.NewReader(`{"cwd":"`+dir+`"}`), &out, compact); err != nil {
		t.Fatalf("hookRun: %v", err)
	}
	var got struct {
		HookSpecificOutput struct {
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out.String())
	}
	return got.HookSpecificOutput.AdditionalContext
}

// The agent sees only the first 2048 bytes of additionalContext, so only the
// items whose index line ends inside them count as surfaced.
func TestHookRun_RecordsOnlyItemsWithinFirst2KB(t *testing.T) {
	for _, compact := range []bool{true, false} {
		t.Run(fmt.Sprintf("compact=%v", compact), func(t *testing.T) {
			store := newTestStore(t)
			dir := newTestRepo(t, testRemote)
			for i := range 40 {
				store.Add(park.Item{Name: fmt.Sprintf("Item %02d %s", i, strings.Repeat("x", 50)), Remote: testRemote})
			}

			ctx := hookContext(t, store, dir, compact)

			_, lines, _ := strings.Cut(ctx, "\n\n")
			offset := len(ctx) - len(lines)
			var visible []int64
			for _, line := range strings.Split(strings.TrimSuffix(lines, "\n"), "\n") {
				offset += len(line)
				if offset > 2048 {
					break
				}
				offset++ // the newline
				id, _ := strconv.ParseInt(strings.TrimPrefix(strings.Fields(line)[0], "#"), 10, 64)
				visible = append(visible, id)
			}
			if len(ctx) <= 2048 || len(visible) == 0 || len(visible) == 40 {
				t.Fatalf("test setup: context is %d bytes with %d visible lines; want a partial fit", len(ctx), len(visible))
			}

			want := usageFor(t, store, "surfaced", "session-hook", visible...)
			if got := usageRows(t, store); !reflect.DeepEqual(got, want) {
				t.Errorf("usage rows = %d, want the %d items within 2048 bytes", len(got), len(want))
			}
		})
	}
}

func TestHookRun_RecordsEveryItemWhenAllFit(t *testing.T) {
	store := newTestStore(t)
	dir := newTestRepo(t, testRemote)
	a, _ := store.Add(park.Item{Name: "one", Remote: testRemote})
	b, _ := store.Add(park.Item{Name: "two", Remote: testRemote})
	store.Add(park.Item{Name: "elsewhere", Remote: "https://github.com/other/repo"})

	hookContext(t, store, dir, true)

	if got, want := usageRows(t, store), usageFor(t, store, "surfaced", "session-hook", a, b); !reflect.DeepEqual(got, want) {
		t.Errorf("usage rows = %q, want %q", got, want)
	}
}

func TestHookRun_SilentWhenNothingParked(t *testing.T) {
	store := newTestStore(t)
	dir := newTestRepo(t, "https://github.com/svandragt/park")

	var out bytes.Buffer
	if err := hookRun(store, strings.NewReader(`{"cwd":"`+dir+`"}`), &out, true); err != nil {
		t.Fatalf("hookRun: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("expected no output, got %q", out.String())
	}
}

func TestHookRun_SilentOutsideGitRepo(t *testing.T) {
	store := newTestStore(t)
	dir := t.TempDir()

	var out bytes.Buffer
	if err := hookRun(store, strings.NewReader(`{"cwd":"`+dir+`"}`), &out, true); err != nil {
		t.Fatalf("hookRun: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("expected no output, got %q", out.String())
	}
}

func TestHookRun_SilentOnGarbagePayload(t *testing.T) {
	store := newTestStore(t)

	var out bytes.Buffer
	if err := hookRun(store, strings.NewReader("not json"), &out, true); err != nil {
		t.Fatalf("hookRun: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("expected no output, got %q", out.String())
	}
}

// sessionStartCommands returns every command string under hooks.SessionStart.
func sessionStartCommands(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var cfg struct {
		Hooks struct {
			SessionStart []struct {
				Hooks []struct {
					Command string `json:"command"`
				} `json:"hooks"`
			} `json:"SessionStart"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse %s: %v\n%s", path, err, data)
	}
	var cmds []string
	for _, entry := range cfg.Hooks.SessionStart {
		for _, h := range entry.Hooks {
			cmds = append(cmds, h.Command)
		}
	}
	return cmds
}

func TestInstallHook_CreatesMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "settings.json")

	added, err := installHook(path, false)
	if err != nil {
		t.Fatalf("installHook: %v", err)
	}
	if !added {
		t.Error("expected added = true for a fresh file")
	}
	if cmds := sessionStartCommands(t, path); len(cmds) != 1 || cmds[0] != hookCommand {
		t.Errorf("commands = %v, want [%s]", cmds, hookCommand)
	}
}

func TestInstallHook_PreservesExistingHooks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	existing := `{
  "model": "opus",
  "hooks": {
    "SessionStart": [
      {"hooks": [{"type": "command", "command": "someone-elses-hook"}]}
    ]
  }
}`
	if err := os.WriteFile(path, []byte(existing), 0600); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if _, err := installHook(path, false); err != nil {
		t.Fatalf("installHook: %v", err)
	}

	cmds := sessionStartCommands(t, path)
	if len(cmds) != 2 || cmds[0] != "someone-elses-hook" || cmds[1] != hookCommand {
		t.Errorf("commands = %v, want [someone-elses-hook %s]", cmds, hookCommand)
	}

	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), `"model"`) {
		t.Errorf("unrelated settings keys were dropped:\n%s", data)
	}
}

func TestInstallHook_Idempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")

	if _, err := installHook(path, false); err != nil {
		t.Fatalf("first install: %v", err)
	}
	added, err := installHook(path, false)
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if added {
		t.Error("expected added = false on second install")
	}
	if cmds := sessionStartCommands(t, path); len(cmds) != 1 {
		t.Errorf("commands = %v, want exactly one entry", cmds)
	}
}

const testRemote = "https://github.com/svandragt/park"

// hookFileOut runs hookFile for a tool call on file in repo dir and returns
// its raw output.
func hookFileOut(t *testing.T, store *park.Store, dir, sessionID, file string) string {
	t.Helper()
	payload, _ := json.Marshal(map[string]any{
		"cwd":        dir,
		"session_id": sessionID,
		"tool_input": map[string]any{"file_path": file},
	})
	var out bytes.Buffer
	if err := hookFile(store, bytes.NewReader(payload), &out); err != nil {
		t.Fatalf("hookFile: %v", err)
	}
	return out.String()
}

// hookFileLines returns the item lines of hookFile's additionalContext, or nil
// when it printed nothing.
func hookFileLines(t *testing.T, raw string) []string {
	t.Helper()
	if raw == "" {
		return nil
	}
	var got struct {
		HookSpecificOutput struct {
			HookEventName     string `json:"hookEventName"`
			AdditionalContext string `json:"additionalContext"`
		} `json:"hookSpecificOutput"`
	}
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, raw)
	}
	if got.HookSpecificOutput.HookEventName != "PostToolUse" {
		t.Errorf("hookEventName = %q, want PostToolUse", got.HookSpecificOutput.HookEventName)
	}
	preamble, lines, ok := strings.Cut(got.HookSpecificOutput.AdditionalContext, "\n\n")
	if !ok || !strings.Contains(preamble, "do not act unprompted") || !strings.Contains(preamble, "park show <id>") {
		t.Errorf("unexpected preamble %q", preamble)
	}
	return strings.Split(strings.TrimSuffix(lines, "\n"), "\n")
}

func TestHookFile_SurfacesItemNamingTheFile(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	store := newTestStore(t)
	dir := newTestRepo(t, testRemote)
	id, _ := store.Add(park.Item{Name: "Leak in the hook", Body: "see cmd/hook.go", Remote: testRemote})
	store.Add(park.Item{Name: "Elsewhere", Body: "see cmd/hook.go", Remote: "https://github.com/other/repo"})
	store.Add(park.Item{Name: "Unrelated", Body: "see cmd/list.go", Remote: testRemote})

	lines := hookFileLines(t, hookFileOut(t, store, dir, "s1", filepath.Join(dir, "cmd", "hook.go")))

	if want := fmt.Sprintf("#%d  Leak in the hook", id); len(lines) != 1 || lines[0] != want {
		t.Errorf("lines = %q, want [%q]", lines, want)
	}
}

func TestHookFile_ShowsEachItemOncePerSession(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	store := newTestStore(t)
	dir := newTestRepo(t, testRemote)
	store.Add(park.Item{Name: "Leak", Body: "cmd/hook.go", Remote: testRemote})
	file := filepath.Join(dir, "cmd", "hook.go")

	if hookFileOut(t, store, dir, "s1", file) == "" {
		t.Fatal("first call printed nothing")
	}
	if got := hookFileOut(t, store, dir, "s1", file); got != "" {
		t.Errorf("second call in the same session printed %q", got)
	}
	if hookFileOut(t, store, dir, "s2", file) == "" {
		t.Error("a new session printed nothing")
	}
	// No session_id means no record to dedupe against.
	for range 2 {
		if hookFileOut(t, store, dir, "", file) == "" {
			t.Error("a call without session_id printed nothing")
		}
	}
}

func TestHookFile_RecordsWhatItPrinted(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	store := newTestStore(t)
	dir := newTestRepo(t, testRemote)
	a, _ := store.Add(park.Item{Name: "One", Body: "cmd/hook.go", Remote: testRemote})
	b, _ := store.Add(park.Item{Name: "Two", Body: "cmd/hook.go", Remote: testRemote})
	store.Add(park.Item{Name: "Unrelated", Body: "cmd/list.go", Remote: testRemote})
	file := filepath.Join(dir, "cmd", "hook.go")

	hookFileOut(t, store, dir, "s1", file)
	hookFileOut(t, store, dir, "s1", file) // dedup: prints nothing, so records nothing

	if got, want := usageRows(t, store), usageFor(t, store, "surfaced", "file-hook", a, b); !reflect.DeepEqual(got, want) {
		t.Errorf("usage rows = %q, want %q", got, want)
	}
}

func TestHookFile_SanitisesSessionID(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	store := newTestStore(t)
	dir := newTestRepo(t, testRemote)
	store.Add(park.Item{Name: "Leak", Body: "cmd/hook.go", Remote: testRemote})

	hookFileOut(t, store, dir, "../evil/id", filepath.Join(dir, "cmd", "hook.go"))

	if _, err := os.Stat(filepath.Join(tmp, "park-hook-evilid")); err != nil {
		t.Errorf("expected a sanitised record file: %v", err)
	}
}

func TestHookFile_FallsBackToBasename(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	store := newTestStore(t)
	dir := newTestRepo(t, testRemote)
	id, _ := store.Add(park.Item{Name: "Moved file", Body: "hook.go used to live in src", Remote: testRemote})

	lines := hookFileLines(t, hookFileOut(t, store, dir, "s1", filepath.Join(dir, "cmd", "hook.go")))

	if want := fmt.Sprintf("#%d  Moved file", id); len(lines) != 1 || lines[0] != want {
		t.Errorf("lines = %q, want [%q]", lines, want)
	}
}

func TestHookFile_DropsBasenameMatchesWhenTooMany(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	store := newTestStore(t)
	dir := newTestRepo(t, testRemote)
	for i := range 6 {
		store.Add(park.Item{Name: fmt.Sprintf("Page %d", i), Body: fmt.Sprintf("pages/p%d/index.php", i), Remote: testRemote})
	}

	if got := hookFileOut(t, store, dir, "s1", filepath.Join(dir, "x", "index.php")); got != "" {
		t.Errorf("expected no output for a basename shared by 6 items, got %q", got)
	}
}

func TestHookFile_ListsAtMostFiveItems(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	store := newTestStore(t)
	dir := newTestRepo(t, testRemote)
	for i := range 7 {
		store.Add(park.Item{Name: fmt.Sprintf("Item %d", i), Body: "cmd/hook.go", Remote: testRemote})
	}

	lines := hookFileLines(t, hookFileOut(t, store, dir, "s1", filepath.Join(dir, "cmd", "hook.go")))

	if len(lines) != 5 {
		t.Errorf("got %d lines, want 5: %q", len(lines), lines)
	}
}

func TestHookFile_SilentOnBadInput(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	store := newTestStore(t)
	dir := newTestRepo(t, testRemote)
	store.Add(park.Item{Name: "Leak", Body: "cmd/hook.go", Remote: testRemote})

	for name, payload := range map[string]string{
		"garbage":           "not json",
		"empty":             "",
		"no file_path":      `{"cwd":"` + dir + `","tool_input":{}}`,
		"no cwd":            `{"tool_input":{"file_path":"` + filepath.Join(dir, "cmd", "hook.go") + `"}}`,
		"outside the repo":  `{"cwd":"` + dir + `","tool_input":{"file_path":"/etc/cmd/hook.go"}}`,
		"relative path":     `{"cwd":"` + dir + `","tool_input":{"file_path":"cmd/hook.go"}}`,
		"cwd not a git dir": `{"cwd":"` + t.TempDir() + `","tool_input":{"file_path":"/x/cmd/hook.go"}}`,
	} {
		var out bytes.Buffer
		if err := hookFile(store, strings.NewReader(payload), &out); err != nil || out.Len() != 0 {
			t.Errorf("%s: err = %v, output = %q; want silence", name, err, out.String())
		}
	}
	if got := usageRows(t, store); len(got) != 0 {
		t.Errorf("bad input recorded usage: %q", got)
	}
}

// settingsHooks returns the matcher and command of every hooks.<event> entry.
func settingsHooks(t *testing.T, path, event string) [][2]string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var cfg struct {
		Hooks map[string][]struct {
			Matcher string `json:"matcher"`
			Hooks   []struct {
				Command string `json:"command"`
			} `json:"hooks"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse %s: %v\n%s", path, err, data)
	}
	var got [][2]string
	for _, entry := range cfg.Hooks[event] {
		for _, h := range entry.Hooks {
			got = append(got, [2]string{entry.Matcher, h.Command})
		}
	}
	return got
}

func TestInstallHook_AddsPostToolUseWhenAsked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")

	added, err := installHook(path, true)
	if err != nil || !added {
		t.Fatalf("installHook = %v, %v; want true, nil", added, err)
	}

	want := [][2]string{{"Read|Edit|Write|MultiEdit", "park hook file"}}
	if got := settingsHooks(t, path, "PostToolUse"); !reflect.DeepEqual(got, want) {
		t.Errorf("PostToolUse = %v, want %v", got, want)
	}
	if cmds := sessionStartCommands(t, path); len(cmds) != 1 || cmds[0] != hookCommand {
		t.Errorf("SessionStart commands = %v, want [%s]", cmds, hookCommand)
	}

	added, err = installHook(path, true)
	if err != nil || added {
		t.Errorf("second install = %v, %v; want false, nil", added, err)
	}
	if got := settingsHooks(t, path, "PostToolUse"); len(got) != 1 {
		t.Errorf("PostToolUse = %v, want exactly one entry", got)
	}
}

// An earlier park already wrote the SessionStart hook; the upgrade still has
// to add the PostToolUse one.
func TestInstallHook_AddsPostToolUseToExistingInstall(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if _, err := installHook(path, false); err != nil {
		t.Fatalf("first install: %v", err)
	}

	added, err := installHook(path, true)
	if err != nil || !added {
		t.Fatalf("upgrade = %v, %v; want true, nil", added, err)
	}
	if got := settingsHooks(t, path, "PostToolUse"); len(got) != 1 {
		t.Errorf("PostToolUse = %v, want one entry", got)
	}
	if cmds := sessionStartCommands(t, path); len(cmds) != 1 {
		t.Errorf("SessionStart commands = %v, want exactly one entry", cmds)
	}
}

func TestInstallHook_SkipsPostToolUseWhenNotAsked(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hooks.json")
	if _, err := installHook(path, false); err != nil {
		t.Fatalf("installHook: %v", err)
	}
	if got := settingsHooks(t, path, "PostToolUse"); len(got) != 0 {
		t.Errorf("PostToolUse = %v, want none", got)
	}
}
