package cmd

import (
	"os"
	"strings"
	"testing"
)

// Plain type names such as "docs" appear in prose, so match the list each doc spells out.
func TestDocsListEveryItemType(t *testing.T) {
	readme, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatal(err)
	}
	if want := "(" + strings.Join(itemTypes, "/") + ")"; !strings.Contains(string(readme), want) {
		t.Errorf("README.md does not list the item types as %s", want)
	}

	skill, err := os.ReadFile("../.claude/skills/park/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, typ := range itemTypes {
		if !strings.Contains(string(skill), "`"+typ+"`") {
			t.Errorf("SKILL.md does not list item type `%s`", typ)
		}
	}
}
