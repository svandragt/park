---
name: park
version: 1.0.0
description: "Park context for later. Use when the user says: 'park this', 'park that', 'save this for later', 'remember this', 'I'll come back to this', 'park the current work'. Also handles: 'what did I park?', 'show parked items', 'list parked', 'unpark', 'done with #N', 'resolve #N'."
author: svandragt
---

# Park Skill

Saves work-in-progress context to a local SQLite DB (`$XDG_DATA_HOME/park/park.db`, defaulting to `~/.local/share/park/park.db`) via the `park` CLI. Override DB path with `PARK_DB` env var for cross-device sync (Syncthing, Dropbox, etc.).

## Rules

1. When the user says "park this/that/something" or wants to save context for later, extract metadata from the current conversation and run `park add`.
2. Never ask for confirmation before parking — extract and run immediately, then report the ID.
3. When the user asks to see what's parked, run `park list`.
4. When the user marks something done/resolved, run `park done <id>`.

## Parking: extract these fields from conversation context

| Flag | What to extract |
|---|---|
| `--name` | Short title — what was being worked on |
| `--desc` | One-line hook that makes this item recognizable later |
| `--body` | The current context: what file/function/problem, current state |
| `--why` | Why this matters / what the end goal is |
| `--how` | Concrete next step to resume from here |
| `--tags` | Optional: relevant keywords (language, area, feature name) |

Git remote, branch, device, and timestamp are auto-detected by the CLI.

## Commands

```bash
# Park current context
park add --name "..." --desc "..." --body "..." --why "..." --how "..."

# List active items (default)
park list

# List all statuses
park list --status all

# Filter by repo
park list --remote https://github.com/org/repo

# Show full detail
park show <id>

# Mark done
park done <id>

# Archive
park archive <id>
```

## Example

User: "park this, I need to go deal with something"

Claude extracts context from the conversation and runs:
```bash
park add \
  --name "Refactoring auth middleware" \
  --desc "Replacing session token storage for compliance" \
  --body "In pkg/auth/middleware.go:47 — replacing cookie store with encrypted JWT. Half-done: handler updated, tests not yet." \
  --why "Legal flagged current session token storage" \
  --how "Finish updating TestAuthMiddleware, then run go test ./pkg/auth/..."
```

Then reports: `parked #3: Refactoring auth middleware`
