package cmd

import "fmt"

const helpText = `park — save and recall work context

Usage:
  park <command> [flags]

Commands:
  add              Park a new item (auto-captures host, remote, branch)
  edit <id>        Update fields on an existing item
  list, ls         List items (filter by --status/--remote/--branch/--tag/--type/--current)
  search <kw>      Full-text search across name/description/body/why/how/tags
  show <id>        Show full detail for one item (use - for most recent)
  done <id>        Mark item resolved (use - for most recent)
  archive <id>     Archive item
  reopen <id>      Move item back to active
  delete <id>      Hard-delete an item
  prune            Hard-delete resolved/archived items older than --days (default 30)
  migrate <dir>    Copy the DB to a new directory
  rename-remote    Bulk-update git_remote across all items (old → new)
  serve            Start a web UI to browse parked items (--addr, default 127.0.0.1:7654)
  help             Show this help (also --help, -h)

Environment:
  PARK_DB          Override the DB path (default: $XDG_DATA_HOME/park/park.db)

Run 'park <command> -h' for command-specific flags.
`

// HelpText returns the top-level help string.
func HelpText() string {
	return helpText
}

// RunHelp prints the top-level help to stdout.
func RunHelp(_ []string) error {
	fmt.Print(HelpText())
	return nil
}
