package park_test

import (
	"testing"

	_ "modernc.org/sqlite"

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

func TestGetLast_Empty(t *testing.T) {
	s := newTestStore(t)
	_, err := s.GetLast()
	if err != park.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDelete_RemovesItem(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "to delete"})

	if err := s.Delete(id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := s.Get(id)
	if err != park.ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	s := newTestStore(t)
	if err := s.Delete(999); err != park.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetLast_ReturnsMostRecent(t *testing.T) {
	s := newTestStore(t)
	s.Add(park.Item{Name: "first"})
	id, _ := s.Add(park.Item{Name: "second"})

	item, err := s.GetLast()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.ID != id {
		t.Errorf("got ID %d, want %d", item.ID, id)
	}
}
