package cmd

import (
	"fmt"

	"github.com/svandragt/park/internal/park"
)

func RunShow(store *park.Store, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: park show <id>")
	}
	id, err := parseID(store, args[0])
	if err != nil {
		return err
	}
	it, err := store.Get(id)
	if err != nil {
		return err
	}

	fmt.Printf("# %s  [#%d · %s]\n\n", it.Name, it.ID, it.Status)
	if it.Description != "" {
		fmt.Printf("**%s**\n\n", it.Description)
	}
	if it.Body != "" {
		fmt.Printf("%s\n\n", it.Body)
	}
	if it.Why != "" {
		fmt.Printf("Why: %s\n", it.Why)
	}
	if it.HowToApply != "" {
		fmt.Printf("How to apply: %s\n", it.HowToApply)
	}
	fmt.Printf("\nType: %s", it.Type)
	if it.Tags != "" {
		fmt.Printf("  Tags: %s", it.Tags)
	}
	if it.Remote != "" {
		fmt.Printf("\nRepo: %s  Branch: %s", it.Remote, it.Branch)
	}
	if it.Parent != "" {
		if parent, err := store.GetByUID(it.Parent); err == nil {
			fmt.Printf("\nParent: #%d %s", parent.ID, parent.Name)
		}
	}
	fmt.Printf("\nDevice: %s  Parked: %s\n", it.Device, it.CreatedAt.Format("2006-01-02 15:04"))

	kids, err := store.Children(it.UID)
	if err != nil {
		return err
	}
	if len(kids) > 0 {
		fmt.Printf("\nChildren: %d/%d done\n\n%s", countDone(kids), len(kids), formatItems(kids))
	}
	return nil
}
