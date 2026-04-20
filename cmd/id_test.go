package cmd

import (
	"testing"

	"github.com/svandragt/park/internal/db"
	"github.com/svandragt/park/internal/park"
)

func newTestStore(t *testing.T) *park.Store {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return park.New(conn)
}

func TestParseID_Numeric(t *testing.T) {
	s := newTestStore(t)
	id, err := parseID(s, "42")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 42 {
		t.Errorf("got %d, want 42", id)
	}
}

func TestParseID_Invalid(t *testing.T) {
	s := newTestStore(t)
	_, err := parseID(s, "abc")
	if err == nil {
		t.Error("expected error for invalid id")
	}
}

func TestParseID_Dash_Empty(t *testing.T) {
	s := newTestStore(t)
	_, err := parseID(s, "-")
	if err != park.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestParseID_Dash_ReturnsLastID(t *testing.T) {
	s := newTestStore(t)
	s.Add(park.Item{Name: "first"})
	lastID, _ := s.Add(park.Item{Name: "second"})

	id, err := parseID(s, "-")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != lastID {
		t.Errorf("got %d, want %d", id, lastID)
	}
}
