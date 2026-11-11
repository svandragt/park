package cmd

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/svandragt/park/internal/db"
	"github.com/svandragt/park/internal/park"
)

// hookCommand is what gets written into the agent config as the SessionStart hook.
const hookCommand = "park hook run"

// hookPreamble frames the item list for the agent. The "do not act on it
// unprompted" clause is load-bearing: without it a parked item reads as a
// to-do list and the agent starts working on it.
const hookPreamble = "Parked work for this project (park CLI). Surface anything relevant to the task at hand; do not act on it unprompted. This list shows names and tags only, so before you start a task, run park search <keywords> and, for a ticket, park list --tag <KEY>. Use park show <id> for detail."

// hookPreambleCompact says the same in fewer bytes, for the compact index.
const hookPreambleCompact = "Parked items here (id name [tags]). Surface anything relevant; do not act on it unprompted. Before a task: park search <words>; for a ticket, park list --tag <KEY>; park show <id> for detail."

// hookFileCommand and hookFileMatcher are the PostToolUse hook that surfaces
// items about a file when the agent touches it. Claude Code only: Codex has no
// matching event.
const (
	hookFileCommand = "park hook file"
	hookFileMatcher = "Read|Edit|Write|MultiEdit"
	hookFileMax     = 5
)

// hookFilePreamble keeps the same "do not act unprompted" clause as hookPreamble.
const hookFilePreamble = "Parked items about this file; surface them if relevant, do not act unprompted. park show <id> for detail."

// hookTargets maps agent name to its config file, relative to the home directory.
// Claude Code and Codex share the same hook schema and wire format.
var hookTargets = map[string]string{
	"claude": ".claude/settings.json",
	"codex":  ".codex/hooks.json",
}

func RunHook(dbPath string, args []string) error {
	if len(args) > 0 && args[0] == "run" {
		return hookRunCLI(dbPath, args[1:])
	}
	if len(args) > 0 && args[0] == "file" {
		return hookFileCLI(dbPath)
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

Claude Code only: to also surface items when the agent reads or edits a file
they name, add:

{
  "hooks": {
    "PostToolUse": [
      { "matcher": %q, "hooks": [ { "type": "command", "command": %q } ] }
    ]
  }
}

Merge into the existing arrays — do not replace them.
Run 'park hook --install' to do this automatically.
`, filepath.Join("~", hookTargets["claude"]), filepath.Join("~", hookTargets["codex"]), hookCommand, hookFileMatcher, hookFileCommand)
}

func installHookFor(agent, settingsPath string) error {
	// --settings alone cannot tell which agent owns the file, so it gets the
	// PostToolUse hook only together with --agent claude.
	if settingsPath != "" {
		return reportInstall(settingsPath, settingsPath, agent == "claude")
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
		return reportInstall(agent, filepath.Join(home, rel), agent == "claude")
	}

	installed := 0
	for _, name := range []string{"claude", "codex"} {
		path := filepath.Join(home, hookTargets[name])
		if _, err := os.Stat(filepath.Dir(path)); err != nil {
			continue // agent not installed on this machine
		}
		if err := reportInstall(name, path, name == "claude"); err != nil {
			return err
		}
		installed++
	}
	if installed == 0 {
		return fmt.Errorf("no agent config directory found; use --agent claude|codex or --settings <path>")
	}
	return nil
}

func reportInstall(name, path string, withFile bool) error {
	added, err := installHook(path, withFile)
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

// installHook merges the SessionStart hook, and with withFile the PostToolUse
// file hook, into the config file at path, preserving every other key and any
// hooks already registered. Each hook is checked on its own, so a config that
// has only the SessionStart hook still gains the other. It reports whether an
// entry was added.
func installHook(path string, withFile bool) (bool, error) {
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

	added := addHookEntry(hooks, string(data), "SessionStart", "", hookCommand)
	if withFile {
		added = addHookEntry(hooks, string(data), "PostToolUse", hookFileMatcher, hookFileCommand) || added
	}
	if !added {
		return false, nil
	}
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

// addHookEntry appends command to hooks[event] unless the config text already
// names it. It reports whether it added the entry.
func addHookEntry(hooks map[string]any, existing, event, matcher, command string) bool {
	if strings.Contains(existing, command) {
		return false
	}
	entry := map[string]any{
		"hooks": []any{map[string]any{"type": "command", "command": command}},
	}
	if matcher != "" {
		entry["matcher"] = matcher
	}
	entries, _ := hooks[event].([]any)
	hooks[event] = append(entries, entry)
	return true
}

// hookRunCLI is the SessionStart hook body. Every failure path is silent: a
// session that opens with a hook error is worse than one with no hook.
func hookRunCLI(dbPath string, args []string) error {
	compact, err := hookRunCompact(args)
	if err != nil {
		return nil
	}
	conn, err := db.Open(dbPath)
	if err != nil {
		return nil
	}
	defer conn.Close()
	return hookRun(park.New(conn), os.Stdin, os.Stdout, compact)
}

// hookRunCompact parses the flags of park hook run.
func hookRunCompact(args []string) (bool, error) {
	fs := flag.NewFlagSet("hook run", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	compact := fs.Bool("compact", true, "drop tags the name repeats and shorten the preamble")
	err := fs.Parse(args)
	return *compact, err
}

// compactTags drops tags that the name or an earlier tag already contains,
// ignoring case. Search still reads the stored tags.
func compactTags(name, tags string) string {
	lowerName := strings.ToLower(name)
	seen := map[string]bool{}
	var keep []string
	for _, t := range strings.Split(tags, ",") {
		t = strings.TrimSpace(t)
		k := strings.ToLower(t)
		if k == "" || seen[k] || strings.Contains(lowerName, k) {
			continue
		}
		seen[k] = true
		keep = append(keep, t)
	}
	return strings.Join(keep, ",")
}

func hookRun(store *park.Store, in io.Reader, out io.Writer, compact bool) error {
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
	// Claude Code shows the agent only the first 2 KB of hook output, so
	// one line per item keeps more items inside it.
	preamble, line, sep := hookPreamble, "#%d  %s", "  "
	if compact {
		preamble, line, sep = hookPreambleCompact, "%d %s", " "
	}
	var index strings.Builder
	for _, it := range items {
		recap += fmt.Sprintf("\n  #%d  %s", it.ID, it.Name)
		fmt.Fprintf(&index, line, it.ID, it.Name)
		tags := it.Tags
		if compact {
			tags = compactTags(it.Name, tags)
		}
		if tags != "" {
			fmt.Fprintf(&index, "%s[%s]", sep, tags)
		}
		index.WriteString("\n")
	}

	return json.NewEncoder(out).Encode(map[string]any{
		"systemMessage": recap,
		"hookSpecificOutput": map[string]any{
			"hookEventName":     "SessionStart",
			"additionalContext": preamble + "\n\n" + index.String(),
		},
	})
}

// hookFileCLI is the PostToolUse hook body. Like hookRunCLI, every failure is silent.
func hookFileCLI(dbPath string) error {
	conn, err := db.Open(dbPath)
	if err != nil {
		return nil
	}
	defer conn.Close()
	return hookFile(park.New(conn), os.Stdin, os.Stdout)
}

// hookFile prints the active items on this repo's remote that name the file
// the agent just touched, each at most once per session.
func hookFile(store *park.Store, in io.Reader, out io.Writer) error {
	var payload struct {
		Cwd       string `json:"cwd"`
		SessionID string `json:"session_id"`
		ToolInput struct {
			FilePath string `json:"file_path"`
		} `json:"tool_input"`
	}
	if err := json.NewDecoder(in).Decode(&payload); err != nil || payload.Cwd == "" || payload.ToolInput.FilePath == "" {
		return nil
	}

	remote := normalizeRemote(gitOutput("-C", payload.Cwd, "remote", "get-url", "origin"))
	top := gitOutput("-C", payload.Cwd, "rev-parse", "--show-toplevel")
	if remote == "" || top == "" {
		return nil
	}
	rel, err := filepath.Rel(top, payload.ToolInput.FilePath)
	if err != nil || !filepath.IsLocal(rel) {
		return nil // relative path, or a file outside the repo
	}
	rel = filepath.ToSlash(rel)

	items, err := store.Mentions(remote, rel)
	if err != nil {
		return nil
	}
	if base := filepath.Base(rel); len(items) == 0 && base != rel {
		// A basename shared by many items (index.php) says nothing about this file.
		if items, err = store.Mentions(remote, base); err != nil || len(items) > hookFileMax {
			return nil
		}
	}

	record := hookShownPath(payload.SessionID)
	items = unshown(items, record)
	if len(items) > hookFileMax {
		items = items[:hookFileMax]
	}
	if len(items) == 0 {
		return nil
	}

	var lines strings.Builder
	for _, it := range items {
		fmt.Fprintf(&lines, "#%d  %s\n", it.ID, it.Name)
	}
	if err := json.NewEncoder(out).Encode(map[string]any{
		"hookSpecificOutput": map[string]any{
			"hookEventName":     "PostToolUse",
			"additionalContext": hookFilePreamble + "\n\n" + lines.String(),
		},
	}); err != nil {
		return nil
	}
	markShown(record, items)
	return nil
}

// hookShownPath is where a session records the item IDs it has been shown. It
// returns "" without a usable session id, which turns the dedup off.
func hookShownPath(sessionID string) string {
	id := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return -1
	}, sessionID)
	if id == "" {
		return ""
	}
	return filepath.Join(os.TempDir(), "park-hook-"+id)
}

// unshown drops the items whose ID the record file already lists.
func unshown(items []park.Item, record string) []park.Item {
	data, err := os.ReadFile(record)
	if record == "" || err != nil {
		return items
	}
	seen := map[string]bool{}
	for _, id := range strings.Fields(string(data)) {
		seen[id] = true
	}
	var keep []park.Item
	for _, it := range items {
		if !seen[strconv.FormatInt(it.ID, 10)] {
			keep = append(keep, it)
		}
	}
	return keep
}

// markShown appends the item IDs to the record file; a failure only means the
// items may show again.
func markShown(record string, items []park.Item) {
	if record == "" {
		return
	}
	f, err := os.OpenFile(record, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	for _, it := range items {
		fmt.Fprintln(f, it.ID)
	}
}
