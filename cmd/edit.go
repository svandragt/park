package cmd

import (
	"flag"
	"fmt"

	"github.com/svandragt/park/internal/park"
)

func RunEdit(store *park.Store, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: park edit <id> [flags]")
	}
	id, err := parseID(store, args[0])
	if err != nil {
		return err
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
	fs.Visit(func(fl *flag.Flag) {
		switch fl.Name {
		case "name":
			f.Name = name
		case "desc":
			f.Description = desc
		case "body":
			f.Body = body
		case "why":
			f.Why = why
		case "how":
			f.HowToApply = how
		case "tags":
			f.Tags = tags
		case "type":
			f.Type = typ
		}
	})

	if err := store.Update(id, f); err != nil {
		return err
	}
	fmt.Printf("#%d updated\n", id)
	return nil
}
