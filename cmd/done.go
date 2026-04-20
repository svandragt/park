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
	if err := store.SetStatus(id, status); err != nil {
		return err
	}
	fmt.Printf("#%d marked as %s\n", id, status)
	return nil
}
