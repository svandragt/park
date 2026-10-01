package cmd

import (
	"fmt"
	"strings"

	"github.com/svandragt/park/internal/park"
)

// resolveParent turns a --parent value (240, #240 or -) into the parent's
// UID, since integer ids are per-device. An empty ref means no parent.
func resolveParent(store *park.Store, ref string) (string, error) {
	if ref == "" {
		return "", nil
	}
	id, err := parseID(store, strings.TrimPrefix(ref, "#"))
	if err != nil {
		return "", err
	}
	parent, err := store.Get(id)
	if err != nil {
		return "", err
	}
	return parent.UID, nil
}

func countDone(items []park.Item) int {
	n := 0
	for _, it := range items {
		if it.Status == "resolved" {
			n++
		}
	}
	return n
}

// printParentHeader prints "#240 v0.4  3/7 done" above a --parent filtered list.
func printParentHeader(store *park.Store, parentUID string) error {
	parent, err := store.GetByUID(parentUID)
	if err != nil {
		return err
	}
	kids, err := store.Children(parentUID)
	if err != nil {
		return err
	}
	fmt.Printf("#%d %s  %d/%d done\n", parent.ID, parent.Name, countDone(kids), len(kids))
	return nil
}

// warnOpenChildren lists the active children of a just-resolved item. It
// never blocks: resolving a milestone early is the user's call.
func warnOpenChildren(store *park.Store, id int64) error {
	it, err := store.Get(id)
	if err != nil {
		return err
	}
	open, err := store.List(park.ListFilter{Parent: it.UID, Status: "active"})
	if err != nil || len(open) == 0 {
		return err
	}
	fmt.Printf("warning: %d open child item(s):\n%s", len(open), formatItems(open))
	return nil
}
