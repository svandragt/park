---
name: park
version: 1.1.0
description: "Park context for later. Use when the user says: 'park this', 'park that', 'save this for later', 'remember this', 'I'll come back to this', 'park the current work'. Also handles: 'what did I park?', 'show parked items', 'list parked', 'unpark', 'done with #N', 'resolve #N', 'work on #N', 'resume #N', 'pick up #N'. Also handles: 'park github issues', 'park existing github issues', 'park open issues'."
author: svandragt
---

# Park Skill

Saves work-in-progress context to a local SQLite DB (`$XDG_DATA_HOME/park/park.db`, defaulting to `~/.local/share/park/park.db`) via the `park` CLI. Override DB path with `PARK_DB` env var for cross-device sync (Syncthing, Dropbox, etc.).

## Rules

1. When the user says "park this/that/something" or wants to save context for later, extract metadata from the current conversation and run `park add`.
2. Never ask for confirmation before parking — extract and run immediately, then report the ID.
3. When the user asks to see what's parked, auto-detect the current git remote (`git remote get-url origin 2>/dev/null`) and pass it via `--remote` to scope results to the current project. If no remote exists, run `park list` without a filter.
4. When the user marks something done/resolved, run `park done <id>`.
5. When the user wants to work on, resume, or unpark an item, run `park show <id>` and present the full context (body, why, how) so they can pick up immediately.

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

## Parking GitHub Issues

**Trigger phrases:** "park github issues", "park existing github issues", "park open issues", "park issues from github"

**Workflow:**

1. Detect the repo remote:
   ```bash
   git remote get-url origin 2>/dev/null
   ```

2. Fetch open issues (respect any label/milestone/limit the user specified):
   ```bash
   gh issue list --state open --json number,title,body,labels,url,milestone --limit 50
   ```

3. For each issue, check for duplicates before parking:
   ```bash
   park search "<issue title>"
   ```
   Skip the issue if a match is found (report it as already parked).

4. For each non-duplicate issue, **reason** about the field mapping — do not dump raw content:

   | Flag | How to populate |
   |---|---|
   | `--name` | Issue title verbatim |
   | `--desc` | One-line summary synthesized from the body (fall back to title if body is empty) |
   | `--body` | Issue body trimmed and reformatted — remove boilerplate, keep the substance |
   | `--why` | Inferred motivation: what problem does this fix, or what value does it add? |
   | `--how` | Concrete first step to start working on this issue |
   | `--tags` | Label names normalized (lowercase, hyphens→underscores) plus any inferred keywords |
   | `--type` | Inferred from labels: `bug`, `feature`, `chore`, `docs` — default `project` |

   ```bash
   park add --name "..." --desc "..." --body "..." --why "..." --how "..." --tags "..." --type "..."
   ```

5. Report a summary: `parked N issues: #id Title, ...` and list any skipped duplicates.

**Conversational filters** (handle without needing explicit flags):
- "park issues labelled bug" → add `--label bug` to the `gh issue list` call
- "park issues in milestone v2" → add `--milestone v2`
- "park the top 5 issues" → set `--limit 5`

**Skip criteria:**
- Issues with no body and a one-word title (too sparse to synthesize meaningful fields)
- Issues already present in park (duplicate check above)
