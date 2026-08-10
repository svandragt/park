package cmd

import (
	"bytes"
	"encoding/json"
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
	if err := hookRun(store, strings.NewReader(`{"cwd":"`+dir+`"}`), &out); err != nil {
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

func TestHookRun_SilentWhenNothingParked(t *testing.T) {
	store := newTestStore(t)
	dir := newTestRepo(t, "https://github.com/svandragt/park")

	var out bytes.Buffer
	if err := hookRun(store, strings.NewReader(`{"cwd":"`+dir+`"}`), &out); err != nil {
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
	if err := hookRun(store, strings.NewReader(`{"cwd":"`+dir+`"}`), &out); err != nil {
		t.Fatalf("hookRun: %v", err)
	}
	if out.Len() != 0 {
		t.Errorf("expected no output, got %q", out.String())
	}
}

func TestHookRun_SilentOnGarbagePayload(t *testing.T) {
	store := newTestStore(t)

	var out bytes.Buffer
	if err := hookRun(store, strings.NewReader("not json"), &out); err != nil {
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
