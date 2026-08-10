package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/svandragt/park/internal/db"
	"github.com/svandragt/park/internal/park"
)

// hookCommand is what gets written into the agent config as the SessionStart hook.
const hookCommand = "park hook run"

// hookPreamble frames the item list for the agent. The "do not act on it
// unprompted" clause is load-bearing: without it a parked item reads as a
// to-do list and the agent starts working on it.
const hookPreamble = "Parked work for this project (park CLI). Surface anything relevant to the task at hand; do not act on it unprompted. Use park show <id> for detail."

// hookTargets maps agent name to its config file, relative to the home directory.
// Claude Code and Codex share the same hook schema and wire format.
var hookTargets = map[string]string{
	"claude": ".claude/settings.json",
	"codex":  ".codex/hooks.json",
}

func RunHook(dbPath string, args []string) error {
	if len(args) > 0 && args[0] == "run" {
		return hookRunCLI(dbPath)
	}

	fs := flag.NewFlagSet("hook", flag.ContinueOnError)
	install := fs.Bool("install", false, "merge the hook into the agent config files")
	agent := fs.String("agent", "", "agent to install for (claude/codex); default: every one found")
	settings := fs.String("settings", "", "config file to install into (overrides --agent)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*install {
		printHookConfig()
		return nil
	}
	return installHookFor(*agent, *settings)
}

func printHookConfig() {
	fmt.Printf(`Add this to your agent config to surface parked items at session start:

  %s  (Claude Code)
  %s  (Codex)

{
  "hooks": {
    "SessionStart": [
      { "hooks": [ { "type": "command", "command": %q } ] }
    ]
  }
}

Merge into the existing SessionStart array — do not replace it.
Run 'park hook --install' to do this automatically.
`, filepath.Join("~", hookTargets["claude"]), filepath.Join("~", hookTargets["codex"]), hookCommand)
}

func installHookFor(agent, settingsPath string) error {
	if settingsPath != "" {
		return reportInstall(settingsPath, settingsPath)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	if agent != "" {
		rel, ok := hookTargets[agent]
		if !ok {
			return fmt.Errorf("unknown agent %q (want claude or codex)", agent)
		}
		return reportInstall(agent, filepath.Join(home, rel))
	}

	installed := 0
	for _, name := range []string{"claude", "codex"} {
		path := filepath.Join(home, hookTargets[name])
		if _, err := os.Stat(filepath.Dir(path)); err != nil {
			continue // agent not installed on this machine
		}
		if err := reportInstall(name, path); err != nil {
			return err
		}
		installed++
	}
	if installed == 0 {
		return fmt.Errorf("no agent config directory found; use --agent claude|codex or --settings <path>")
	}
	return nil
}

func reportInstall(name, path string) error {
	added, err := installHook(path)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if !added {
		fmt.Printf("%s: already installed (%s)\n", name, path)
		return nil
	}
	fmt.Printf("%s: hook installed in %s\n", name, path)
	fmt.Println("  Takes effect next session (/hooks forces a reload).")
	return nil
}

// installHook merges the SessionStart hook into the config file at path,
// preserving every other key and any hooks already registered. It reports
// whether an entry was added.
func installHook(path string) (bool, error) {
	cfg := map[string]any{}
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, &cfg); err != nil {
			return false, fmt.Errorf("parse %s: %w", path, err)
		}
	case !os.IsNotExist(err):
		return false, err
	}

	hooks, _ := cfg["hooks"].(map[string]any)
	if hooks == nil {
		hooks = map[string]any{}
	}
	entries, _ := hooks["SessionStart"].([]any)

	if strings.Contains(string(data), hookCommand) {
		return false, nil
	}

	hooks["SessionStart"] = append(entries, map[string]any{
		"hooks": []any{map[string]any{"type": "command", "command": hookCommand}},
	})
	cfg["hooks"] = hooks

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return false, err
	}
	tmp := path + ".park-tmp"
	if err := os.WriteFile(tmp, append(out, '\n'), 0600); err != nil {
		return false, err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return false, err
	}
	return true, nil
}

// hookRunCLI is the SessionStart hook body. Every failure path is silent: a
// session that opens with a hook error is worse than one with no hook.
func hookRunCLI(dbPath string) error {
	conn, err := db.Open(dbPath)
	if err != nil {
		return nil
	}
	defer conn.Close()
	return hookRun(park.New(conn), os.Stdin, os.Stdout)
}

func hookRun(store *park.Store, in io.Reader, out io.Writer) error {
	var payload struct {
		Cwd string `json:"cwd"`
	}
	// The hook does not necessarily run in the session's directory, so the
	// payload's cwd is the only reliable source. A wrong directory silently
	// yields another repo's items, which looks identical to "nothing parked".
	if err := json.NewDecoder(in).Decode(&payload); err != nil || payload.Cwd == "" {
		return nil
	}

	remote := normalizeRemote(gitOutput("-C", payload.Cwd, "remote", "get-url", "origin"))
	if remote == "" {
		return nil
	}

	items, err := store.List(park.ListFilter{Status: "active", Remote: remote})
	if err != nil || len(items) == 0 {
		return nil
	}

	// systemMessage is the only field the user sees; additionalContext reaches
	// the agent silently.
	recap := fmt.Sprintf("park: %d parked item(s) here", len(items))
	for _, it := range items {
		recap += fmt.Sprintf("\n  #%d  %s", it.ID, it.Name)
	}

	return json.NewEncoder(out).Encode(map[string]any{
		"systemMessage": recap,
		"hookSpecificOutput": map[string]any{
			"hookEventName":     "SessionStart",
			"additionalContext": hookPreamble + "\n\n" + formatItems(items),
		},
	})
}
