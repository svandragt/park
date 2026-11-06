package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	added, err := installHook(path)
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

	if _, err := installHook(path); err != nil {
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

	if _, err := installHook(path); err != nil {
		t.Fatalf("first install: %v", err)
	}
	added, err := installHook(path)
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
