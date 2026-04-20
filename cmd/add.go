package cmd

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/svandragt/park/internal/park"
)

func RunAdd(store *park.Store, args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	name := fs.String("name", "", "title of the parked item (required)")
	desc := fs.String("desc", "", "one-line hook / description")
	body := fs.String("body", "", "full context")
	why := fs.String("why", "", "why this matters")
	how := fs.String("how", "", "how to apply / pick up from here")
	tags := fs.String("tags", "", "comma-separated tags")
	typ := fs.String("type", "project", "item type")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *name == "" {
		return fmt.Errorf("--name is required")
	}

	device, _ := os.Hostname()
	rawRemote := gitOutput("remote", "get-url", "origin")
	rawRemote = strings.TrimSuffix(rawRemote, ".git")
	remote := resolveRemote(rawRemote)
	if remote != rawRemote && rawRemote != "" {
		if n, err := store.UpdateRemote(rawRemote, remote); err == nil && n > 0 {
			fmt.Printf("remote renamed: %s → %s (%d item(s) updated)\n", rawRemote, remote, n)
		}
	}
	branch := gitOutput("branch", "--show-current")

	id, err := store.Add(park.Item{
		Name:        *name,
		Description: *desc,
		Type:        *typ,
		Body:        *body,
		Why:         *why,
		HowToApply:  *how,
		Tags:        *tags,
		GitRemote:   remote,
		Branch:      branch,
		Device:      device,
	})
	if err != nil {
		return err
	}
	fmt.Printf("parked #%d: %s\n", id, *name)
	return nil
}

func gitOutput(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
