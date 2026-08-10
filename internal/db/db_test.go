package db

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// An existing database on the old schema is renamed to `remote`, keeping its rows.
func TestMigrate_RenamesGitRemoteColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := conn.Exec(`
CREATE TABLE parks (
	id           INTEGER PRIMARY KEY,
	name         TEXT NOT NULL,
	description  TEXT NOT NULL DEFAULT '',
	type         TEXT NOT NULL DEFAULT 'project',
	body         TEXT NOT NULL DEFAULT '',
	why          TEXT NOT NULL DEFAULT '',
	how_to_apply TEXT NOT NULL DEFAULT '',
	git_remote   TEXT NOT NULL DEFAULT '',
	branch       TEXT NOT NULL DEFAULT '',
	tags         TEXT NOT NULL DEFAULT '',
	status       TEXT NOT NULL DEFAULT 'active',
	device       TEXT NOT NULL DEFAULT '',
	created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);
INSERT INTO parks (name, git_remote) VALUES ('x', 'https://example.com/r');
`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	conn.Close()

	conn, err = Open(path)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	defer conn.Close()

	var got string
	if err := conn.QueryRow(`SELECT remote FROM parks WHERE name = 'x'`).Scan(&got); err != nil {
		t.Fatalf("select remote: %v", err)
	}
	if got != "https://example.com/r" {
		t.Errorf("remote = %q, want the seeded URL", got)
	}
}

// A fresh database gets `remote` directly, and migrating twice is a no-op.
func TestMigrate_FreshAndRepeated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.db")
	conn, err := Open(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	conn.Close()

	conn, err = Open(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer conn.Close()

	var n int
	if err := conn.QueryRow(`SELECT count(*) FROM pragma_table_info('parks') WHERE name = 'remote'`).Scan(&n); err != nil {
		t.Fatalf("pragma: %v", err)
	}
	if n != 1 {
		t.Errorf("remote column count = %d, want 1", n)
	}
}
