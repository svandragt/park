package cmd

import (
	"flag"
	"fmt"
	"time"

	"github.com/svandragt/park/internal/park"
)

func RunPrune(store *park.Store, args []string) error {
	fs := flag.NewFlagSet("prune", flag.ContinueOnError)
	days := fs.Int("days", 30, "delete resolved/archived items older than this many days")
	if err := fs.Parse(args); err != nil {
		return err
	}
	before := time.Now().AddDate(0, 0, -*days)
	n, err := store.Prune(before)
	if err != nil {
		return err
	}
	fmt.Printf("%d item(s) pruned\n", n)
	return nil
}
