# park

A CLI tool for saving and recalling human work context, backed by SQLite. (private alternative for GitHub Issues)

**You're mid-debug when a colleague asks for urgent help.** Park where you are, context-switch, come back and pick up exactly where you left off — right file, right line, right next step.

**You're juggling three repos.** `park list` shows everything active across all of them. `park list --remote github.com/org/repo` scopes it to one.

**You context-switch between machines.** Set `PARK_SYNC_DIR` to a synced folder and your parked items follow you — without the database itself ever being shared. See [Sync across machines](#sync-across-machines).

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
park edit <id> --append-body "new notes"   # adds to the body instead of replacing it
park edit <id> --status resolved   # equivalent to park done <id>
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
park sync-seed --i-understand-this-runs-once  # one-time: seed the sync log from this machine
park rebuild --yes                 # rebuild the local database from the sync logs
park serve                         # browse items in a web UI (default 127.0.0.1:7654)
park serve --addr :7654            # listen on a different address
park version                       # print the version (also --version, -v)
park help                          # show usage (also --help, -h)
```

`add` automatically captures hostname, git remote, and current git branch.
If the remote has been renamed (e.g. a GitHub repo rename), `add` detects the
redirect and updates all existing items to the canonical URL automatically.

## Configuration

| Variable | Default |
|---|---|
| `PARK_DB` | `~/.local/share/park/park.db` (XDG-aware) |
| `PARK_SYNC_DIR` | unset; when set, park writes and reads sync logs there |

## Sync across machines

To use park on more than one machine, give each machine its own log file in a
shared folder. Set `PARK_SYNC_DIR` to a folder your file syncer keeps in step,
and leave `PARK_DB` on its local default:

```bash
export PARK_DB="$HOME/.local/share/park/park.db"
export PARK_SYNC_DIR="$HOME/sync/park"
```

Each machine appends its changes to `<hostname>.jsonl` and never writes another
machine's file, so the syncer has no conflicts to resolve. On each run, park
folds every log it can see into the local database. Every machine ends up with
the full set of items.

Set this up once, on the machine that already holds your items:

```bash
park sync-seed --i-understand-this-runs-once
```

Run it on exactly one machine. It writes every existing item to that machine's
log. A second run, or a run on a second machine, duplicates every item under a
new identifier. The command refuses if this machine's log already has entries.

On every other machine, build the local database from the logs:

```bash
park rebuild --yes
```

`rebuild` discards the local database and rebuilds it from the logs alone, so
run it only when the logs hold everything you want to keep.

When two machines change the same item, the later change wins. Items are
matched by a generated identifier, not by the number you see in `park list`,
which stays local to each machine.

### Do not put the database in a synced folder

Earlier versions suggested pointing `PARK_DB` at a synced folder. Don't.
SQLite writes a database as several files that must stay in step, and a file
syncer copies them one at a time. A half-copied set fails to open with
`database disk image is malformed`, and turning off write-ahead logging
narrows that window without closing it.

`PARK_SYNC_DIR` avoids the problem: a log file is append-only and has exactly
one writer, which every file syncer handles safely.

## Claude Code skill

A skill for [Claude Code](https://claude.ai/code) is included in the repository. It lets you park and resume context using natural language ("park this", "show parked", "work on #2").

To install, copy the skill into Claude's skills directory:

```bash
mkdir -p ~/.claude/skills/park
cp .claude/skills/park/SKILL.md ~/.claude/skills/park/SKILL.md
```

Once installed, Claude will recognize park-related phrases and call the `park` CLI automatically.

### Surface parked items at session start

`park list` only gets run when you already suspect something is parked. To make the
agent tell you instead, install a `SessionStart` hook:

```bash
park hook --install
```

This adds `park hook run` to `~/.claude/settings.json` (Claude Code) and
`~/.codex/hooks.json` (Codex), for whichever of the two exists. Both agents use the same
hook format. Run `park hook` on its own to print the config block without writing anything,
or pass `--agent claude`, `--agent codex`, or `--settings <path>` to target one file.

At the start of each session, `park hook run` reads the session's working directory,
resolves its git remote, and injects that repo's active items as session context. You also
see a short summary of the parked items, so you know the hook ran. It
prints nothing when there is no remote, no database, or nothing parked — a session that
opens with a hook error is worse than one with no hook.

The installer merges into any existing `SessionStart` hooks rather than replacing them.
Hooks load at session start, so the change takes effect next session; in Claude Code,
`/hooks` forces a reload.

## Web UI

To browse parked items in a browser instead of the terminal, run:

```bash
park serve
```

This starts a local server at `http://127.0.0.1:7654` with full-text search, a
repo filter, and clickable tags and types. Pass `--addr` to change the listen
address. The server binds to localhost by default, so the UI stays private to
your machine.

The UI is read-only, so `park serve` copies the database once at startup and
serves that snapshot. It never holds the live database open, and it shows the
items as they were when the server started.

## Build

```bash
go build ./...
go test ./...
```

To cut a release, see [RELEASING.md](RELEASING.md).
