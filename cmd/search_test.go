package cmd

import (
	"reflect"
	"testing"

	"github.com/svandragt/park/internal/park"
)

func TestRunSearch_FlagsAfterKeyword(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "foo"})
	_ = s.SetStatus(id, "resolved")

	if err := RunSearch(s, []string{"foo", "--status", "all"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunSearch_RecordsSurfacedItems(t *testing.T) {
	s := newTestStore(t)
	a, _ := s.Add(park.Item{Name: "widget one"})
	b, _ := s.Add(park.Item{Name: "widget two"})
	s.Add(park.Item{Name: "gadget"})

	captureStdout(t, func() error { return RunSearch(s, []string{"widget"}) })
	captureStdout(t, func() error { return RunSearch(s, []string{"nothing-matches-this"}) })

	if got, want := usageRows(t, s), usageFor(t, s, "surfaced", "search", a, b); !reflect.DeepEqual(got, want) {
		t.Errorf("usage rows = %q, want %q", got, want)
	}
}
