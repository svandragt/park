package cmd

import (
	"reflect"
	"testing"

	"github.com/svandragt/park/internal/park"
)

// Only a --tag list is a lookup; a bare list is a bulk listing and records nothing.
func TestRunList_RecordsSurfacedItemsOnlyWithTag(t *testing.T) {
	s := newTestStore(t)
	tagged, _ := s.Add(park.Item{Name: "tagged", Tags: "BH-1"})
	s.Add(park.Item{Name: "other", Tags: "BH-2"})

	captureStdout(t, func() error { return RunList(s, nil) })
	if got := usageRows(t, s); len(got) != 0 {
		t.Fatalf("a list without --tag recorded usage: %q", got)
	}

	captureStdout(t, func() error { return RunList(s, []string{"--tag", "BH-1"}) })
	captureStdout(t, func() error { return RunList(s, []string{"--tag", "BH-404"}) })

	if got, want := usageRows(t, s), usageFor(t, s, "surfaced", "list-tag", tagged); !reflect.DeepEqual(got, want) {
		t.Errorf("usage rows = %q, want %q", got, want)
	}
}
