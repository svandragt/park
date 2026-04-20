# park

A CLI tool for saving and recalling work context, backed by SQLite.

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
```

`add` automatically captures hostname, git remote, and current branch.

## Configuration

| Variable | Default |
|---|---|
| `PARK_DB` | `~/.local/share/park/park.db` (XDG-aware) |

## Build

```bash
go build ./...
go test ./...
```
