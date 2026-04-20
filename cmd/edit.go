package cmd

import (
	"flag"
	"fmt"
	"strconv"

	"github.com/svandragt/park/internal/park"
)

func RunEdit(store *park.Store, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: park edit <id> [flags]")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid id: %s", args[0])
	}

	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	name := fs.String("name", "", "new title")
	desc := fs.String("desc", "", "new description")
	body := fs.String("body", "", "new body")
	why := fs.String("why", "", "new why")
	how := fs.String("how", "", "new how-to-apply")
	tags := fs.String("tags", "", "new tags")
	typ := fs.String("type", "", "new type")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	f := park.UpdateFields{}
	if *name != "" {
		f.Name = name
	}
	if *desc != "" {
		f.Description = desc
	}
	if *body != "" {
		f.Body = body
	}
	if *why != "" {
		f.Why = why
	}
	if *how != "" {
		f.HowToApply = how
	}
	if *tags != "" {
		f.Tags = tags
	}
	if *typ != "" {
		f.Type = typ
	}

	if err := store.Update(id, f); err != nil {
		return err
	}
	fmt.Printf("#%d updated\n", id)
	return nil
}
