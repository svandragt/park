package cmd

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/svandragt/park/internal/db"
	"github.com/svandragt/park/internal/park"
)

func newSeedTestStore(t *testing.T) *park.Store {
	t.Helper()
	conn, err := db.Open(":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return park.New(conn)
}

func TestRunSyncSeed_RequiresSyncDir(t *testing.T) {
	s := newSeedTestStore(t)
	if err := RunSyncSeed(s, "", []string{"--i-understand-this-runs-once"}); err == nil {
		t.Error("expected error when PARK_SYNC_DIR is unset")
	}
}

func TestRunSyncSeed_RefusesWithoutFlag(t *testing.T) {
	s := newSeedTestStore(t)
	s.Add(park.Item{Name: "a"})
	dir := t.TempDir()

	if err := RunSyncSeed(s, dir, nil); err == nil {
		t.Error("expected error without --i-understand-this-runs-once")
	}
}

func TestRunSyncSeed_RefusesWhenLogAlreadySeeded(t *testing.T) {
	s := newSeedTestStore(t)
	s.Add(park.Item{Name: "a"})
	dir := t.TempDir()
	device, _ := os.Hostname()
	if err := os.WriteFile(filepath.Join(dir, device+".jsonl"), []byte(`{"uid":"x"}`+"\n"), 0600); err != nil {
		t.Fatalf("seed existing log: %v", err)
	}

	if err := RunSyncSeed(s, dir, []string{"--i-understand-this-runs-once"}); err == nil {
		t.Error("expected error when this device's log already has entries")
	}
}

func TestRunSyncSeed_WritesOneAddEventPerRow(t *testing.T) {
	s := newSeedTestStore(t)
	s.Add(park.Item{Name: "a"})
	s.Add(park.Item{Name: "b"})
	dir := t.TempDir()

	if err := RunSyncSeed(s, dir, []string{"--i-understand-this-runs-once"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	device, _ := os.Hostname()
	f, err := os.Open(filepath.Join(dir, device+".jsonl"))
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer f.Close()
	n := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		n++
	}
	if n != 2 {
		t.Errorf("expected 2 seeded events, got %d", n)
	}
}
