package park_test

import (
	"testing"

	"github.com/svandragt/park/internal/park"
	"github.com/svandragt/park/internal/synclog"
)

func TestFold_AppliesAddEditStatusDelete(t *testing.T) {
	s := newTestStore(t)
	dir := t.TempDir()

	uid := synclog.NewULID()
	events := []synclog.Event{
		{UID: uid, Op: "add", TS: "2026-01-01T00:00:00Z", Device: "host1", Fields: map[string]any{
			"name": "original", "status": "active",
		}},
		{UID: uid, Op: "edit", TS: "2026-01-01T00:00:01Z", Device: "host1", Fields: map[string]any{
			"name": "renamed",
		}},
		{UID: uid, Op: "status", TS: "2026-01-01T00:00:02Z", Device: "host1", Fields: map[string]any{
			"status": "resolved",
		}},
	}
	for _, ev := range events {
		if err := synclog.Append(dir, "host1", ev); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	applied, err := s.Fold(dir)
	if err != nil {
		t.Fatalf("fold: %v", err)
	}
	if applied != 3 {
		t.Fatalf("applied = %d, want 3", applied)
	}

	items, err := s.List(park.ListFilter{Status: "resolved"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 resolved item, got %d", len(items))
	}
	if items[0].Name != "renamed" {
		t.Errorf("name = %q, want renamed", items[0].Name)
	}
	if items[0].UID != uid {
		t.Errorf("uid = %q, want %q", items[0].UID, uid)
	}
}

func TestFold_AddIsIdempotent(t *testing.T) {
	s := newTestStore(t)
	dir := t.TempDir()
	uid := synclog.NewULID()
	ev := synclog.Event{UID: uid, Op: "add", TS: "2026-01-01T00:00:00Z", Device: "host1", Fields: map[string]any{"name": "a"}}
	if err := synclog.Append(dir, "host1", ev); err != nil {
		t.Fatalf("append: %v", err)
	}

	if _, err := s.Fold(dir); err != nil {
		t.Fatalf("first fold: %v", err)
	}
	if _, err := s.Fold(dir); err != nil {
		t.Fatalf("second fold: %v", err)
	}

	items, err := s.List(park.ListFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item after re-folding, got %d", len(items))
	}
}

func TestFold_DeleteRemovesRow(t *testing.T) {
	s := newTestStore(t)
	dir := t.TempDir()
	uid := synclog.NewULID()
	if err := synclog.Append(dir, "host1", synclog.Event{UID: uid, Op: "add", TS: "2026-01-01T00:00:00Z", Fields: map[string]any{"name": "a"}}); err != nil {
		t.Fatalf("append add: %v", err)
	}
	if err := synclog.Append(dir, "host1", synclog.Event{UID: uid, Op: "delete", TS: "2026-01-01T00:00:01Z"}); err != nil {
		t.Fatalf("append delete: %v", err)
	}

	if _, err := s.Fold(dir); err != nil {
		t.Fatalf("fold: %v", err)
	}

	items, err := s.List(park.ListFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected item deleted, got %d items", len(items))
	}
}

// An edit/status/delete for a uid whose "add" hasn't arrived yet is dropped
// silently rather than queued, a documented ceiling.
func TestFold_SkipsEventsForUnknownUID(t *testing.T) {
	s := newTestStore(t)
	dir := t.TempDir()
	if err := synclog.Append(dir, "host1", synclog.Event{UID: "unknown", Op: "status", TS: "2026-01-01T00:00:00Z", Fields: map[string]any{"status": "resolved"}}); err != nil {
		t.Fatalf("append: %v", err)
	}

	applied, err := s.Fold(dir)
	if err != nil {
		t.Fatalf("fold: %v", err)
	}
	if applied != 0 {
		t.Fatalf("applied = %d, want 0", applied)
	}
}

// Cross-machine ordering: an "edit" from host2 that came after host1's "add"
// in time must apply even though host2's file is folded independently.
func TestFold_OrdersEventsAcrossFilesByTimestamp(t *testing.T) {
	s := newTestStore(t)
	dir := t.TempDir()
	uid := synclog.NewULID()
	if err := synclog.Append(dir, "host1", synclog.Event{UID: uid, Op: "add", TS: "2026-01-01T00:00:00Z", Fields: map[string]any{"name": "a"}}); err != nil {
		t.Fatalf("append add: %v", err)
	}
	if err := synclog.Append(dir, "host2", synclog.Event{UID: uid, Op: "edit", TS: "2026-01-01T00:00:01Z", Fields: map[string]any{"name": "b"}}); err != nil {
		t.Fatalf("append edit: %v", err)
	}

	if _, err := s.Fold(dir); err != nil {
		t.Fatalf("fold: %v", err)
	}

	item, err := s.Get(1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if item.Name != "b" {
		t.Errorf("name = %q, want b (edit should apply after add)", item.Name)
	}
}

// The fold must not re-emit what it reads: applying events via the emitting
// paths would make the log grow without bound on every fold.
func TestFold_EmitsNothing(t *testing.T) {
	s := newTestStore(t)
	dir := t.TempDir()
	uid1, uid2 := synclog.NewULID(), synclog.NewULID()
	events := []synclog.Event{
		{UID: uid1, Op: "add", TS: "2026-01-01T00:00:00Z", Fields: map[string]any{"name": "a"}},
		{UID: uid2, Op: "add", TS: "2026-01-01T00:00:01Z", Fields: map[string]any{"name": "b"}},
		{UID: uid1, Op: "status", TS: "2026-01-01T00:00:02Z", Fields: map[string]any{"status": "resolved"}},
	}
	for _, ev := range events {
		if err := synclog.Append(dir, "host1", ev); err != nil {
			t.Fatalf("append: %v", err)
		}
	}

	sink := &fakeSink{}
	s.SetSink(sink)

	if _, err := s.Fold(dir); err != nil {
		t.Fatalf("fold: %v", err)
	}

	if len(sink.events) != 0 {
		t.Fatalf("expected fold to emit nothing, sink saw %d events: %+v", len(sink.events), sink.events)
	}
}

func TestFold_PersistsOffsetsSoRefoldIsIncremental(t *testing.T) {
	s := newTestStore(t)
	dir := t.TempDir()
	uid := synclog.NewULID()
	if err := synclog.Append(dir, "host1", synclog.Event{UID: uid, Op: "add", TS: "2026-01-01T00:00:00Z", Fields: map[string]any{"name": "a"}}); err != nil {
		t.Fatalf("append: %v", err)
	}

	applied, err := s.Fold(dir)
	if err != nil {
		t.Fatalf("first fold: %v", err)
	}
	if applied != 1 {
		t.Fatalf("applied = %d, want 1", applied)
	}

	applied, err = s.Fold(dir)
	if err != nil {
		t.Fatalf("second fold: %v", err)
	}
	if applied != 0 {
		t.Fatalf("second fold applied = %d, want 0 (offset should have advanced)", applied)
	}
}
