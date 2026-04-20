package cmd

import (
	"flag"
	"fmt"

	"github.com/svandragt/park/internal/park"
)

func RunList(store *park.Store, args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	status := fs.String("status", "active", "filter by status (active/resolved/archived/all)")
	remote := fs.String("remote", "", "filter by git remote URL")
	branch := fs.String("branch", "", "filter by branch name")
	tag := fs.String("tag", "", "filter by tag")

	if err := fs.Parse(args); err != nil {
		return err
	}

	filterStatus := *status
	if filterStatus == "all" {
		filterStatus = ""
	}

	items, err := store.List(park.ListFilter{
		Status: filterStatus,
		Remote: normalizeURL(*remote),
		Branch: *branch,
		Tag:    *tag,
	})
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Println("no parked items")
		return nil
	}
	for _, it := range items {
		fmt.Printf("#%d  [%s]  %s\n", it.ID, it.Status, it.Name)
		if it.Description != "" {
			fmt.Printf("     %s\n", it.Description)
		}
		if it.GitRemote != "" {
			fmt.Printf("     %s  (%s)\n", it.GitRemote, it.Branch)
		}
		fmt.Println()
	}
	return nil
}
