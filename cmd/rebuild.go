package cmd

import (
	"flag"
	"fmt"

	"github.com/svandragt/park/internal/park"
)

// RunRebuild discards every local row not backed by the shared log and
// folds the log from scratch. This is the command another machine runs to
// adopt the shared dataset, so it requires an explicit --yes: it drops any
// row that only exists locally (never made it into a log).
func RunRebuild(store *park.Store, syncDir string, args []string) error {
	if syncDir == "" {
		return fmt.Errorf("PARK_SYNC_DIR is not set; rebuild needs a log directory")
	}

	fs := flag.NewFlagSet("rebuild", flag.ContinueOnError)
	confirm := fs.Bool("yes", false, "required: acknowledge this discards local rows not present in the shared log")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*confirm {
		return fmt.Errorf("rebuild discards every local row and rebuilds from %s; pass --yes to confirm", syncDir)
	}

	dropped, err := store.ResetLocal()
	if err != nil {
		return err
	}

	applied, err := store.Fold(syncDir)
	if err != nil {
		return err
	}

	fmt.Printf("dropped %d local row(s), folded %d event(s) back in\n", dropped, applied)
	return nil
}
