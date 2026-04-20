package cmd

import (
	"fmt"

	"github.com/svandragt/park/internal/park"
)

func RunDelete(store *park.Store, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: park delete <id>")
	}
	id, err := parseID(store, args[0])
	if err != nil {
		return err
	}
	if err := store.Delete(id); err != nil {
		return err
	}
	fmt.Printf("#%d deleted\n", id)
	return nil
}
