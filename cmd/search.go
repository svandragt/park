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
	typ := fs.String("type", "", "filter by type (task/project/bug/feature/chore/docs)")
	parentRef := fs.String("parent", "", "filter by parent item id (#240, 240 or -)")
	current := fs.Bool("current", false, "filter by current git remote and branch")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("usage: park search [--status all] [--remote URL] [--branch NAME] [--tag TAG] [--type TYPE] [--parent ID] [--current] <keyword>")
	}
	keyword := strings.Join(fs.Args(), " ")

	if *current {
		*remote = currentRemote()
		*branch = currentBranch()
	}

	parent, err := resolveParent(store, *parentRef)
	if err != nil {
		return err
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
		Parent: parent,
	})
	if err != nil {
		return err
	}
	if parent != "" {
		if err := printParentHeader(store, parent); err != nil {
			return err
		}
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
		if it.Remote != "" {
			fmt.Printf("     %s  (%s)\n", it.Remote, it.Branch)
		}
		fmt.Println()
	}
	return nil
}
