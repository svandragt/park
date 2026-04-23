# park

A CLI tool for saving and recalling human work context, backed by SQLite. (private alternative for GitHub Issues)

**You're mid-debug when a colleague asks for urgent help.** Park where you are, context-switch, come back and pick up exactly where you left off — right file, right line, right next step.

**You're juggling three repos.** `park list` shows everything active across all of them. `park list --remote github.com/org/repo` scopes it to one.

**You context-switch between machines.** Point `PARK_DB` at a synced folder (Syncthing, Dropbox) and your parked items follow you.

**Your AI assistant's context window is filling up, or you switch between multiple assistants.** Park the current task so the next session can pick up exactly where you left off, without re-explaining everything.

## How is park different?

| Tool | What it does | What park adds |
|---|---|---|
| **git stash** | Saves uncommitted code changes in the current repo | Saves *why* you were there, the next step, and context across *all* repos — no code changes required |
| **LLM session resume** | Re-feeds prior conversation to the AI | A structured, queryable record you write once; the next session starts from facts, not a transcript |
| **GitHub Issues** | Tracks bugs and features for a team | Local-first and private; captures device, branch, and "how to pick up" automatically; spans repos without a remote |
| **Todo list** | Records tasks to complete | Records the full context — where in the code, why it matters, exact next step — not just the task name |

## Install

```bash
go install github.com/svandragt/park@latest
```

## Usage

```bash
park add --name "fix auth bug" --desc "Session token issue" --body "..." --why "Blocks release" --how "Start at auth.go:42"
park edit <id> --body "updated context" --tags "auth,urgent"
park list                          # active items
park list --current                # scope to current git remote + branch
park list --status resolved        # resolved items
park list --remote github.com/org/repo  # filter by repo (SSH or HTTPS format)
park list --branch main            # filter by branch
park list --tag auth               # filter by tag
park list --type bug               # filter by type (project/bug/feature/chore/docs)
park search "JWT"                  # full-text search (porter stemming, active items only)
park search --status all "JWT"     # search across all statuses
park search --tag auth "token"     # search within a tag
park search --type bug "crash"     # search within a type
park search --remote github.com/org/repo "fix"  # search within a repo
park search --current "token"      # search in current git remote + branch
park show <id>                     # full detail
park show -                        # show most recently added item
park done <id>                     # mark resolved
park done -                        # resolve most recently added item
park archive <id>                  # archive
park reopen <id>                   # move back to active
park delete <id>                   # hard-delete an item
park prune --days 30               # hard-delete resolved/archived items older than N days
park migrate <dest-dir>            # copy DB to new location, print PARK_DB export line
park rename-remote <old> <new>     # update remote URL across all items
```

`add` automatically captures hostname, remote, and current branch. Both git and
[jj](https://docs.jj-vcs.dev/latest/) repos are supported: git is tried first, with a
jj fallback (nearest bookmark for branch, `jj git remote list` for remote) so
non-colocated jj repos and jj changes without a git branch also get populated.
If the remote has been renamed (e.g. a GitHub repo rename), `add` detects the
redirect and updates all existing items to the canonical URL automatically.

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
