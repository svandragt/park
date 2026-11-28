package cmd

import (
	"fmt"

	"github.com/svandragt/park/internal/park"
)

func RunDone(store *park.Store, args []string) error {
	return setStatus(store, args, "resolved")
}

func RunArchive(store *park.Store, args []string) error {
	return setStatus(store, args, "archived")
}

func setStatus(store *park.Store, args []string, status string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: park %s <id>", status)
	}
	id, err := parseID(store, args[0])
	if err != nil {
		return err
	}
	prev, err := store.Get(id)
	if err != nil {
		return err
	}
	if err := store.SetStatus(id, status); err != nil {
		return err
	}
	fmt.Printf("#%d marked as %s\n", id, status)
	recordClose(store, prev, status)
	if status == "resolved" {
		return warnOpenChildren(store, id)
	}
	return nil
}

// recordClose logs a usage row when an item changes to resolved or archived.
func recordClose(store *park.Store, prev *park.Item, to string) {
	if (to != "resolved" && to != "archived") || prev.Status == to {
		return
	}
	_ = store.RecordUsage([]string{prev.UID}, to, "cli")
}
