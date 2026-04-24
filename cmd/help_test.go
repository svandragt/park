package cmd

import (
	"strings"
	"testing"
)

func TestHelpText_ListsAllSubcommands(t *testing.T) {
	out := HelpText()
	for _, name := range []string{
		"add", "edit", "list", "search", "show",
		"done", "archive", "reopen", "delete",
		"prune", "migrate", "rename-remote", "help",
	} {
		if !strings.Contains(out, name) {
			t.Errorf("HelpText() missing subcommand %q", name)
		}
	}
}

func TestHelpText_MentionsPARK_DB(t *testing.T) {
	if !strings.Contains(HelpText(), "PARK_DB") {
		t.Error("HelpText() should mention PARK_DB env var")
	}
}
