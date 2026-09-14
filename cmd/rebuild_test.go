package cmd

import (
	"testing"

	"github.com/svandragt/park/internal/park"
	"github.com/svandragt/park/internal/synclog"
)

func TestRunRebuild_RequiresSyncDir(t *testing.T) {
	s := newSeedTestStore(t)
	if err := RunRebuild(s, "", []string{"--yes"}); err == nil {
		t.Error("expected error when PARK_SYNC_DIR is unset")
	}
}

func TestRunRebuild_RequiresYesFlag(t *testing.T) {
	s := newSeedTestStore(t)
	dir := t.TempDir()
	if err := RunRebuild(s, dir, nil); err == nil {
		t.Error("expected error without --yes")
	}
}

func TestRunRebuild_DiscardsLocalRowsAndFoldsFromLog(t *testing.T) {
	s := newSeedTestStore(t)
	// A local-only row not present in any log; rebuild must drop it.
	s.Add(park.Item{Name: "local only"})

	dir := t.TempDir()
	uid := synclog.NewULID()
	if err := synclog.Append(dir, "host1", synclog.Event{
		UID: uid, Op: "add", TS: "2026-01-01T00:00:00Z",
		Fields: map[string]any{"name": "from log"},
	}); err != nil {
		t.Fatalf("append: %v", err)
	}

	if err := RunRebuild(s, dir, []string{"--yes"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	items, err := s.List(park.ListFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item after rebuild, got %d", len(items))
	}
	if items[0].Name != "from log" {
		t.Errorf("name = %q, want %q", items[0].Name, "from log")
	}
	if items[0].UID != uid {
		t.Errorf("uid = %q, want %q", items[0].UID, uid)
	}
}
