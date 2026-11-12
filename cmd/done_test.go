package cmd

import (
	"reflect"
	"testing"

	"github.com/svandragt/park/internal/park"
)

func TestRunDone_RecordsResolved(t *testing.T) {
	s := newTestStore(t)
	done, _ := s.Add(park.Item{Name: "finish me"})
	archived, _ := s.Add(park.Item{Name: "shelve me"})

	captureStdout(t, func() error { return RunDone(s, []string{"1"}) })
	captureStdout(t, func() error { return RunArchive(s, []string{"2"}) })

	if done != 1 || archived != 2 {
		t.Fatalf("test setup: ids %d, %d", done, archived)
	}
	if got, want := usageRows(t, s), usageFor(t, s, "resolved", "cli", done); !reflect.DeepEqual(got, want) {
		t.Errorf("usage rows = %q, want %q", got, want)
	}
}
