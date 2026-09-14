package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/svandragt/park/internal/db"
	"github.com/svandragt/park/internal/park"
)

func TestSnapshot_CopiesRowsWithNoWAL(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "park.db")
	conn, err := db.Open(srcPath)
	if err != nil {
		t.Fatalf("open src: %v", err)
	}
	defer conn.Close()
	store := park.New(conn)
	if _, err := store.Add(park.Item{Name: "snapshot-item"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	path, err := snapshot(store)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	defer os.Remove(path)

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected snapshot file at %s: %v", path, err)
	}
	if _, err := os.Stat(path + "-wal"); !os.IsNotExist(err) {
		t.Errorf("expected no -wal file beside snapshot, stat err = %v", err)
	}

	snapConn, err := db.Open(path)
	if err != nil {
		t.Fatalf("open snapshot: %v", err)
	}
	defer snapConn.Close()
	snapStore := park.New(snapConn)

	items, err := snapStore.List(park.ListFilter{})
	if err != nil {
		t.Fatalf("list snapshot: %v", err)
	}
	if len(items) != 1 || items[0].Name != "snapshot-item" {
		t.Errorf("expected snapshot to contain seeded row, got %+v", items)
	}
}
