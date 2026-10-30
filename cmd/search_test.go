package cmd

import (
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
