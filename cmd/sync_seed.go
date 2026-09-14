package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/svandragt/park/internal/park"
	"github.com/svandragt/park/internal/synclog"
)

// RunSyncSeed is a one-time bootstrap: it writes every existing row as an
// "add" event into this device's log. Running it on more than one machine
// duplicates every item under a fresh uid each time, so it refuses unless
// told explicitly and unless this device hasn't already seeded.
func RunSyncSeed(store *park.Store, syncDir string, args []string) error {
	if syncDir == "" {
		return fmt.Errorf("PARK_SYNC_DIR is not set; sync-seed needs a log directory")
	}

	fs := flag.NewFlagSet("sync-seed", flag.ContinueOnError)
	confirm := fs.Bool("i-understand-this-runs-once", false, "required: acknowledge this is a one-time, single-machine bootstrap")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if !*confirm {
		return fmt.Errorf("sync-seed writes every item into this device's log and must run on exactly ONE machine; pass --i-understand-this-runs-once to confirm")
	}

	device, err := os.Hostname()
	if err != nil {
		return err
	}
	logPath := synclog.LogPath(syncDir, device)
	if info, err := os.Stat(logPath); err == nil && info.Size() > 0 {
		return fmt.Errorf("this device's log (%s) already has entries; seeding again would duplicate every item", logPath)
	}

	items, err := store.List(park.ListFilter{})
	if err != nil {
		return err
	}

	sink := synclog.FileSink{Dir: syncDir, Device: device}
	for _, it := range items {
		ev := synclog.Event{
			UID: it.UID,
			Op:  "add",
			TS:  it.CreatedAt.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
			Fields: map[string]any{
				"name": it.Name, "description": it.Description, "type": it.Type,
				"body": it.Body, "why": it.Why, "how_to_apply": it.HowToApply,
				"remote": it.Remote, "branch": it.Branch, "tags": it.Tags,
				"status": it.Status, "device": it.Device,
			},
		}
		if err := sink.Emit(ev); err != nil {
			return err
		}
	}
	fmt.Printf("seeded %d item(s) into %s\n", len(items), logPath)
	return nil
}
