package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/svandragt/park/internal/park"
)

func RunAdd(store *park.Store, args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	name := fs.String("name", "", "title of the parked item (required)")
	desc := fs.String("desc", "", "one-line hook / description")
	body := fs.String("body", "", "full context")
	why := fs.String("why", "", "why this matters")
	how := fs.String("how", "", "how to apply / pick up from here")
	tags := fs.String("tags", "", "comma-separated tags")
	typ := fs.String("type", "task", "item type")
	parentRef := fs.String("parent", "", "parent item id (#240, 240 or - for most recent)")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("--name is required")
	}

	parent, err := resolveParent(store, *parentRef)
	if err != nil {
		return err
	}

	device, _ := os.Hostname()
	rawRemote := currentRemote()
	rawRemote = strings.TrimSuffix(rawRemote, ".git")
	remote := resolveRemote(rawRemote)
	if remote != rawRemote && rawRemote != "" {
		if n, err := store.UpdateRemote(rawRemote, remote); err == nil && n > 0 {
			fmt.Printf("remote renamed: %s → %s (%d item(s) updated)\n", rawRemote, remote, n)
		}
	}
	branch := currentBranch()

	id, err := store.Add(park.Item{
		Name:        *name,
		Description: *desc,
		Type:        *typ,
		Body:        *body,
		Why:         *why,
		HowToApply:  *how,
		Tags:        *tags,
		Remote:      remote,
		Branch:      branch,
		Device:      device,
		Parent:      parent,
	})
	if err != nil {
		return err
	}
	fmt.Printf("parked #%d: %s\n", id, *name)
	return nil
}
