package park_test

import (
	"testing"
	"time"

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

func TestPrune_DeletesOldInactiveItems(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "old resolved"})
	s.SetStatus(id, "resolved")
	s.DB().Exec(`UPDATE parks SET updated_at = datetime('now', '-10 days') WHERE id = ?`, id)

	keep, _ := s.Add(park.Item{Name: "recent resolved"})
	s.SetStatus(keep, "resolved")

	active, _ := s.Add(park.Item{Name: "old active"})
	s.DB().Exec(`UPDATE parks SET updated_at = datetime('now', '-10 days') WHERE id = ?`, active)

	n, err := s.Prune(time.Now().AddDate(0, 0, -7))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 pruned, got %d", n)
	}
	if _, err := s.Get(id); err != park.ErrNotFound {
		t.Errorf("old resolved item should be deleted")
	}
	if _, err := s.Get(keep); err != nil {
		t.Errorf("recent resolved item should survive")
	}
	if _, err := s.Get(active); err != nil {
		t.Errorf("active item should survive even if old")
	}
}

func TestReopen_SetsStatusActive(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "to reopen"})
	s.SetStatus(id, "resolved")

	if err := s.SetStatus(id, "active"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	item, _ := s.Get(id)
	if item.Status != "active" {
		t.Errorf("expected active, got %s", item.Status)
	}
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
