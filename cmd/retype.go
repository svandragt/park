package cmd

import (
	"fmt"

	"github.com/svandragt/park/internal/park"
)

func RunRetype(store *park.Store, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: park retype <old-type> <new-type>")
	}
	n, err := store.RetypeAll(args[0], args[1])
	if err != nil {
		return err
	}
	fmt.Printf("retyped %d item(s): %s → %s\n", n, args[0], args[1])
	return nil
}
