package cmd

import (
	"flag"
	"fmt"
	"strings"

	"github.com/svandragt/park/internal/park"
)

func RunSearch(store *park.Store, args []string) error {
	fs := flag.NewFlagSet("search", flag.ContinueOnError)
	status := fs.String("status", "active", "filter by status (active/resolved/archived/all)")
	remote := fs.String("remote", "", "filter by git remote URL")
	branch := fs.String("branch", "", "filter by branch name")
	tag := fs.String("tag", "", "filter by tag")
	typ := fs.String("type", "", "filter by type (project/bug/feature/chore/docs)")
	current := fs.Bool("current", false, "filter by current git remote and branch")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: park search [--status all] [--remote URL] [--branch NAME] [--tag TAG] [--type TYPE] [--current] <keyword>")
	}
	keyword := strings.Join(fs.Args(), " ")

	if *current {
		*remote = currentRemote()
		*branch = currentBranch()
	}

	filterStatus := *status
	if filterStatus == "all" {
		filterStatus = ""
	}

	items, err := store.Search(keyword, park.ListFilter{
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
		fmt.Println("no results")
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
