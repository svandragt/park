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
- **`internal/park`** — `Store` wraps `*sql.DB`; all SQL lives here (`Add`, `List`, `Get`, `SetStatus`)
- **`cmd/`** — one file per subcommand (`add`, `list`, `show`, `done`/`archive`); each `Run*` function parses its own flags

### Subcommands

| Command | Action |
|---|---|
| `add` | Insert a new item; auto-captures hostname, `git remote`, and current branch |
| `edit <id>` | Update fields on an existing item (`--name`, `--desc`, `--body`, `--why`, `--how`, `--tags`, `--type`) |
| `list` / `ls` | List items filtered by `--status`, `--remote`, `--branch`, `--tag` (default status: `active`) |
| `search <keyword>` | Full-text search across name, description, body, why, how-to-apply, tags (FTS5, porter stemming) |
| `show <id>` | Full detail view of one item |
| `done <id>` | Set status → `resolved` |
| `archive <id>` | Set status → `archived` |
| `rename-remote <old> <new>` | Bulk-update `git_remote` across all items |

### Item statuses

`active` → `resolved` (done) or `archived`

### No flags library beyond stdlib

Uses only `flag.FlagSet` from the standard library — no cobra/urfave.
