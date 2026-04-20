# park

A CLI tool for saving and recalling work context, backed by SQLite.

**You're mid-debug when a colleague asks for urgent help.** Park where you are, context-switch, come back and pick up exactly where you left off — right file, right line, right next step.

**You're juggling three repos.** `park list` shows everything active across all of them. `park list --remote github.com/org/repo` scopes it to one.

**You context-switch between machines.** Point `PARK_DB` at a synced folder (Syncthing, Dropbox) and your parked items follow you.

## Install

```bash
go install github.com/svandragt/park@latest
```

## Usage

```bash
park add --name "fix auth bug" --desc "Session token issue" --body "..." --why "Blocks release" --how "Start at auth.go:42"
park list                          # active items
park list --status resolved        # resolved items
park list --remote github.com/org/repo  # filter by repo
park show <id>                     # full detail
park done <id>                     # mark resolved
park archive <id>                  # archive
park rename-remote <old> <new>     # update remote URL across all items
```

`add` automatically captures hostname, git remote, and current branch. If the
remote has been renamed (e.g. a GitHub repo rename), `add` detects the redirect
and updates all existing items to the canonical URL automatically.

## Configuration

| Variable | Default |
|---|---|
| `PARK_DB` | `~/.local/share/park/park.db` (XDG-aware) |

## Claude Code skill

A skill for [Claude Code](https://claude.ai/code) is included in the repository. It lets you park and resume context using natural language ("park this", "show parked", "work on #2").

To install, copy the skill into Claude's skills directory:

```bash
mkdir -p ~/.claude/skills/park
cp .claude/skills/park/SKILL.md ~/.claude/skills/park/SKILL.md
```

Once installed, Claude will recognize park-related phrases and call the `park` CLI automatically.

## Build

```bash
go build ./...
go test ./...
```
