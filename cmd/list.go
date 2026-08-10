package cmd

import (
	"flag"
	"fmt"
	"strings"

	"github.com/svandragt/park/internal/park"
)

func RunList(store *park.Store, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	status := fs.String("status", "active", "filter by status (active/resolved/archived/all)")
	remote := fs.String("remote", "", "filter by git remote URL")
	branch := fs.String("branch", "", "filter by branch name")
	tag := fs.String("tag", "", "filter by tag")
	typ := fs.String("type", "", "filter by type (project/bug/feature/chore/docs)")
	current := fs.Bool("current", false, "filter by current git remote and branch")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *current {
		*remote = currentRemote()
		*branch = currentBranch()
	}

	filterStatus := *status
	if filterStatus == "all" {
		filterStatus = ""
	}

	items, err := store.List(park.ListFilter{
		Status: filterStatus,
		Remote: normalizeRemote(*remote),
		Branch: *branch,
		Tag:    *tag,
		Type:   *typ,
	})
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Println("no parked items")
		return nil
	}
	fmt.Print(formatItems(items))
	return nil
}

// formatItems renders the one-per-item summary used by list and the session hook.
func formatItems(items []park.Item) string {
	var b strings.Builder
	for _, it := range items {
		fmt.Fprintf(&b, "#%d  [%s]  %s\n", it.ID, it.Status, it.Name)
		if it.Description != "" {
			fmt.Fprintf(&b, "     %s\n", it.Description)
		}
		if it.Remote != "" {
			fmt.Fprintf(&b, "     %s  (%s)\n", it.Remote, it.Branch)
		}
		if it.Tags != "" {
			fmt.Fprintf(&b, "     tags: %s\n", it.Tags)
		}
		b.WriteString("\n")
	}
	return b.String()
}
