package cmd

import (
	"fmt"
	"strings"

	"github.com/svandragt/park/internal/park"
)

func RunRenameRemote(store *park.Store, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: park rename-remote <old-url> <new-url>")
	}
	oldURL := strings.TrimSuffix(args[0], ".git")
	newURL := strings.TrimSuffix(args[1], ".git")

	n, err := store.UpdateRemote(oldURL, newURL)
	if err != nil {
		return err
	}
	fmt.Printf("updated %d item(s): %s → %s\n", n, oldURL, newURL)
	return nil
}
