package park_test

import (
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/svandragt/park/internal/db"
	"github.com/svandragt/park/internal/park"
	"github.com/svandragt/park/internal/synclog"
)

var errSinkBroken = errors.New("sink broken")

type fakeSink struct {
	events []synclog.Event
	err    error
}

func (f *fakeSink) Emit(ev synclog.Event) error {
	f.events = append(f.events, ev)
	return f.err
}

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

func TestSearch_FilterByTag(t *testing.T) {
	s := newTestStore(t)
	s.Add(park.Item{Name: "tagged", Tags: "auth,urgent"})
	s.Add(park.Item{Name: "untagged"})

	results, err := s.Search("tagged", park.ListFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result without tag filter, got %d", len(results))
	}

	results, err = s.Search("tagged", park.ListFilter{Tag: "auth"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Name != "tagged" {
		t.Errorf("expected 1 match for tag=auth, got %d", len(results))
	}

	results, err = s.Search("tagged", park.ListFilter{Tag: "missing"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for non-matching tag, got %d", len(results))
	}
}

func TestSearch_FilterByType(t *testing.T) {
	s := newTestStore(t)
	s.Add(park.Item{Name: "a bug item", Type: "bug"})
	s.Add(park.Item{Name: "a feature item", Type: "feature"})

	results, err := s.Search("item", park.ListFilter{Type: "bug"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Type != "bug" {
		t.Errorf("expected 1 bug result, got %d", len(results))
	}
}

func TestSearch_FilterByRemote(t *testing.T) {
	s := newTestStore(t)
	s.Add(park.Item{Name: "repo item", Remote: "https://github.com/org/repo"})
	s.Add(park.Item{Name: "other item", Remote: "https://github.com/org/other"})

	results, err := s.Search("item", park.ListFilter{Remote: "https://github.com/org/repo"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Remote != "https://github.com/org/repo" {
		t.Errorf("expected 1 result for remote filter, got %d", len(results))
	}
}

func TestSearch_FilterByBranch(t *testing.T) {
	s := newTestStore(t)
	s.Add(park.Item{Name: "main branch item", Branch: "main"})
	s.Add(park.Item{Name: "feat branch item", Branch: "feature/x"})

	results, err := s.Search("item", park.ListFilter{Branch: "main"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Branch != "main" {
		t.Errorf("expected 1 result for branch filter, got %d", len(results))
	}
}

func TestSearch_FindsNewlyAddedItem(t *testing.T) {
	s := newTestStore(t)
	s.Add(park.Item{Name: "widget frobnicator"})

	results, err := s.Search("frobnicator", park.ListFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
}

func TestSearch_EditUpdatesIndex(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "oldname"})

	newName := "renameditem"
	if err := s.Update(id, park.UpdateFields{Name: &newName}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, err := s.Search("renameditem", park.ListFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result for new name, got %d", len(results))
	}

	results, err = s.Search("oldname", park.ListFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for old name, got %d", len(results))
	}
}

func TestSearch_DeleteRemovesFromIndex(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "gonesoon"})

	if err := s.Delete(id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, err := s.Search("gonesoon", park.ListFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results after delete, got %d", len(results))
	}
}

func TestSearch_SetStatusKeepsItemFindable(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "statuschangeitem"})

	if err := s.SetStatus(id, "resolved"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	results, err := s.Search("statuschangeitem", park.ListFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result after status change, got %d", len(results))
	}
}

func TestAdd_GeneratesUID(t *testing.T) {
	s := newTestStore(t)
	id, err := s.Add(park.Item{Name: "has uid"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	item, err := s.Get(id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if item.UID == "" {
		t.Error("expected a generated UID, got empty string")
	}
}

func TestAdd_EmitsAddEvent(t *testing.T) {
	s := newTestStore(t)
	sink := &fakeSink{}
	s.SetSink(sink)

	id, err := s.Add(park.Item{Name: "emits", Tags: "x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sink.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(sink.events))
	}
	ev := sink.events[0]
	if ev.Op != "add" {
		t.Errorf("op = %q, want add", ev.Op)
	}
	item, _ := s.Get(id)
	if ev.UID != item.UID {
		t.Errorf("event uid = %q, want %q", ev.UID, item.UID)
	}
	if ev.Fields["name"] != "emits" {
		t.Errorf("expected fields to carry name, got %+v", ev.Fields)
	}
}

func TestUpdate_EmitsEditEventWithOnlyChangedFields(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "orig"})
	sink := &fakeSink{}
	s.SetSink(sink)

	newName := "renamed"
	if err := s.Update(id, park.UpdateFields{Name: &newName}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sink.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(sink.events))
	}
	ev := sink.events[0]
	if ev.Op != "edit" {
		t.Errorf("op = %q, want edit", ev.Op)
	}
	if ev.Fields["name"] != "renamed" {
		t.Errorf("expected changed name field, got %+v", ev.Fields)
	}
	if _, ok := ev.Fields["description"]; ok {
		t.Errorf("expected unchanged fields to be absent, got %+v", ev.Fields)
	}
}

func TestSetStatus_EmitsStatusEvent(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "orig"})
	sink := &fakeSink{}
	s.SetSink(sink)

	if err := s.SetStatus(id, "resolved"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sink.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(sink.events))
	}
	ev := sink.events[0]
	if ev.Op != "status" {
		t.Errorf("op = %q, want status", ev.Op)
	}
	if ev.Fields["status"] != "resolved" {
		t.Errorf("expected status field, got %+v", ev.Fields)
	}
}

func TestDelete_EmitsDeleteEvent(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "orig"})
	item, _ := s.Get(id)
	sink := &fakeSink{}
	s.SetSink(sink)

	if err := s.Delete(id); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sink.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(sink.events))
	}
	ev := sink.events[0]
	if ev.Op != "delete" {
		t.Errorf("op = %q, want delete", ev.Op)
	}
	if ev.UID != item.UID {
		t.Errorf("event uid = %q, want %q", ev.UID, item.UID)
	}
	if ev.Fields != nil {
		t.Errorf("expected no fields on delete, got %+v", ev.Fields)
	}
}

func TestSinkError_DoesNotFailTheCommand(t *testing.T) {
	s := newTestStore(t)
	sink := &fakeSink{err: errSinkBroken}
	s.SetSink(sink)

	if _, err := s.Add(park.Item{Name: "still works"}); err != nil {
		t.Fatalf("sink error leaked into Add: %v", err)
	}
}

func TestPrune_EmitsDeleteEventPerRow(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "old resolved"})
	s.SetStatus(id, "resolved")
	s.DB().Exec(`UPDATE parks SET updated_at = datetime('now', '-10 days') WHERE id = ?`, id)
	item, _ := s.Get(id)

	sink := &fakeSink{}
	s.SetSink(sink)

	n, err := s.Prune(time.Now().AddDate(0, 0, -7))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 pruned, got %d", n)
	}
	if len(sink.events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(sink.events))
	}
	if sink.events[0].Op != "delete" || sink.events[0].UID != item.UID {
		t.Errorf("unexpected event: %+v", sink.events[0])
	}
}

func TestUpdateRemote_EmitsEditEventPerRow(t *testing.T) {
	s := newTestStore(t)
	s.Add(park.Item{Name: "a", Remote: "https://old"})
	s.Add(park.Item{Name: "b", Remote: "https://old"})

	sink := &fakeSink{}
	s.SetSink(sink)

	n, err := s.UpdateRemote("https://old", "https://new")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 updated, got %d", n)
	}
	if len(sink.events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(sink.events))
	}
	for _, ev := range sink.events {
		if ev.Op != "edit" || ev.Fields["remote"] != "https://new" {
			t.Errorf("unexpected event: %+v", ev)
		}
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
