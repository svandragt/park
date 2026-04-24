# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build ./...          # build
go run . <subcommand>   # run without installing
go test ./...           # all tests
go vet ./...            # static analysis
go install .            # install binary to $GOPATH/bin
```

Set `PARK_DB=/path/to/park.db` to override the default database location (`~/.local/share/park/park.db`).

## Architecture

`park` is a CLI tool for saving and recalling work context, backed by a single SQLite database.

- **`main.go`** — resolves DB path (env `PARK_DB` → XDG → `~/.local/share/park/park.db`), opens DB, dispatches subcommands
- **`internal/db`** — opens the SQLite connection (WAL mode, foreign keys on) and runs the schema migration inline
- **`internal/park`** — `Store` wraps `*sql.DB`; all SQL lives here (`Add`, `List`, `Get`, `SetStatus`, `Delete`, `Prune`)
- **`cmd/`** — one file per subcommand (`add`, `list`, `show`, `done`/`archive`/`reopen`, `delete`, `prune`, `migrate`); each `Run*` function parses its own flags
- **`cmd/vcs.go`** — VCS-neutral `currentRemote` / `currentBranch` helpers; tries git first (`cmd/git.go`) then jj (`cmd/jj.go`). Branch fallback uses `jj log -r 'heads(::@ & bookmarks())'` to pick the nearest bookmark; remote fallback parses `jj git remote list`

### Subcommands

| Command | Action |
|---|---|
| `add` | Insert a new item; auto-captures hostname, remote, and current branch (git first, jj bookmark fallback) |
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
| `rename-remote <old> <new>` | Bulk-update `git_remote` across all items |
| `help` / `--help` / `-h` | Print top-level usage; handled before DB open |

### Item statuses

`active` → `resolved` (done) or `archived`

### No flags library beyond stdlib

Uses only `flag.FlagSet` from the standard library — no cobra/urfave.

## graphify

This project has a graphify-rs knowledge graph at graphify-out/.

Rules:
- Before answering architecture or codebase questions, read graphify-out/GRAPH_REPORT.md for god nodes and community structure
- If graphify-out/wiki/index.md exists, navigate it instead of reading raw files
- After modifying code files in this session, run `graphify-rs build --path . --output graphify-out --no-llm --update` to keep the graph current (fast, AST-only, ~2-5s)
