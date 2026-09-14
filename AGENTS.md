# AGENTS.md

This file provides guidance to coding agents when working with code in this repository.

## Commands

```bash
go build ./...          # build
go run . <subcommand>   # run without installing
go test ./...           # all tests
go vet ./...            # static analysis
go install .            # install binary to $GOPATH/bin
```

Set `PARK_DB` to override the default database location (`~/.local/share/park/park.db`).
Set `PARK_SYNC_DIR` to enable multi-machine sync; leave it unset for a single machine.

## Architecture

`park` is a CLI tool for saving and recalling work context, backed by a local SQLite database.

- **`main.go`** — resolves DB path (env `PARK_DB` → XDG → `~/.local/share/park/park.db`), opens DB, wires the sync sink and folds pending log events when `PARK_SYNC_DIR` is set, dispatches subcommands
- **`internal/db`** — opens the SQLite connection (WAL mode, foreign keys on) and runs the schema migrations inline, each guarded by a row in `migrations`
- **`internal/park`** — `Store` wraps `*sql.DB`; all item SQL lives here. `fold.go` holds the sync read path
- **`internal/synclog`** — the append-only event log: ULID generation, the `Event` type, and reading and writing `<device>.jsonl`
- **`cmd/`** — one file per subcommand; each `Run*` function parses its own flags
- **`cmd/vcs.go`** — `currentRemote` / `currentBranch` helpers over `gitOutput` (`cmd/git.go`)
- **`cmd/serve.go`** — HTTP handlers and the embedded web UI for `park serve`

### Subcommands

| Command | Action |
|---|---|
| `add` | Insert a new item; auto-captures hostname, git remote, and current git branch |
| `edit <id>` | Update fields on an existing item (`--name`, `--desc`, `--body`, `--why`, `--how`, `--tags`, `--type`, `--status`) |
| `list` / `ls` | List items filtered by `--status`, `--remote`, `--branch`, `--tag`, `--type` (default status: `active`); shows tags inline |
| `search <keyword>` | Full-text search across name, description, body, why, how-to-apply, tags (FTS5, porter stemming); supports `--status`, `--remote`, `--branch`, `--tag`, `--type`, `--current` filters (default status: `active`) |
| `show <id>` | Full detail view of one item |
| `done <id>` | Set status → `resolved` |
| `archive <id>` | Set status → `archived` |
| `reopen <id>` | Set status → `active` (reverse done/archive) |
| `delete <id>` | Hard-delete an item from the database |
| `prune` | Hard-delete resolved/archived items older than `--days` (default 30) |
| `migrate <dest-dir>` | Copy DB to a new directory and print the `PARK_DB` export line |
| `rename-remote <old> <new>` | Bulk-update `remote` across all items |
| `sync-seed` | One-time bootstrap: write every existing item to this device's log. Runs on exactly ONE machine; refuses without `--i-understand-this-runs-once` or if the log already has entries |
| `rebuild` | Drop the local items and offsets, then rebuild from the sync logs alone. Requires `--yes` |
| `serve` | Start a local web UI (`--addr`, default `127.0.0.1:7654`) over a startup snapshot of the database |
| `hook` | Print the `SessionStart` hook config; `--install` merges it into the agent settings files. `hook run` is the hook body: reads the payload on stdin, prints this repo's active items as session context plus a user-visible `systemMessage` summary, silent on every failure |
| `help` / `--help` / `-h` | Print top-level usage; handled before DB open |

### Item statuses

`active` → `resolved` (done) or `archived`

### Sync model

A shared SQLite file cannot survive a file syncer, so park never shares one. Each
machine appends its writes to its own `<device>.jsonl` under `PARK_SYNC_DIR` and reads
every log it finds, folding the events into a local database that is safe to delete and
rebuild. No file has two writers, so the syncer has nothing to reconcile.

Things to preserve when changing this code:

- **The fold must not emit.** `Store.Fold` applies events with direct SQL. Routing them
  through `Add`/`Update`/`SetStatus`/`Delete` would re-append everything it just read to
  this device's own log, growing it without bound. `TestFold_EmitsNothing` guards this.
- **Events apply in timestamp order across all logs**, not file by file: an edit from one
  machine can reference an item added on another.
- **A log may be delivered mid-append.** Readers stop at the last newline and leave the
  offset there.
- **`uid` is the identity; the integer id is a local display handle.** Never match items
  across machines by id, and never quote an id anywhere off-machine.
- **An item's `device` is where it was parked**, which is not the same as the machine that
  wrote the log line. Keep them distinct when adding event fields.

### Known ceilings

- An `edit`, `status` or `delete` whose `add` has not arrived is dropped, not queued.
- Logs are never compacted. The data is small enough that this has not mattered.

### No flags library beyond stdlib

Uses only `flag.FlagSet` from the standard library — no cobra/urfave.

## graphify

This project has a graphify-rs knowledge graph at graphify-out/.

Rules:
- Before answering architecture or codebase questions, read graphify-out/GRAPH_REPORT.md for god nodes and community structure
- If graphify-out/wiki/index.md exists, navigate it instead of reading raw files
- After modifying code files in this session, run `graphify-rs build --path . --output graphify-out --no-llm --update` to keep the graph current (fast, AST-only, ~2-5s)
