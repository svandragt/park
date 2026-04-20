package cmd

import (
	"fmt"
	"strings"

	"github.com/svandragt/park/internal/park"
)

func RunSearch(store *park.Store, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: park search <keyword>")
	}
	keyword := strings.Join(args, " ")

	items, err := store.Search(keyword)
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
