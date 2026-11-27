package cmd

import (
	"reflect"
	"sort"
	"testing"

	"github.com/svandragt/park/internal/park"
)

func TestRunDoneAndArchive_RecordUsage(t *testing.T) {
	s := newTestStore(t)
	done, _ := s.Add(park.Item{Name: "finish me"})
	archived, _ := s.Add(park.Item{Name: "shelve me"})

	captureStdout(t, func() error { return RunDone(s, []string{"1"}) })
	captureStdout(t, func() error { return RunArchive(s, []string{"2"}) })

	if done != 1 || archived != 2 {
		t.Fatalf("test setup: ids %d, %d", done, archived)
	}
	want := append(usageFor(t, s, "resolved", "cli", done), usageFor(t, s, "archived", "cli", archived)...)
	sort.Strings(want)
	if got := usageRows(t, s); !reflect.DeepEqual(got, want) {
		t.Errorf("usage rows = %q, want %q", got, want)
	}
}

func TestRunArchive_RecordsArchived(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "shelve me"})

	captureStdout(t, func() error { return RunArchive(s, []string{"1"}) })

	if got, want := usageRows(t, s), usageFor(t, s, "archived", "cli", id); !reflect.DeepEqual(got, want) {
		t.Errorf("usage rows = %q, want %q", got, want)
	}
}

func TestRunDoneAndArchive_NoRowWhenStatusUnchanged(t *testing.T) {
	s := newTestStore(t)
	a, _ := s.Add(park.Item{Name: "already resolved"})
	b, _ := s.Add(park.Item{Name: "already archived"})
	_ = s.SetStatus(a, "resolved")
	_ = s.SetStatus(b, "archived")

	captureStdout(t, func() error { return RunDone(s, []string{"1"}) })
	captureStdout(t, func() error { return RunArchive(s, []string{"2"}) })

	if got := usageRows(t, s); len(got) != 0 {
		t.Errorf("usage rows = %q, want none", got)
	}
}
