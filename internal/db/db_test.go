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

// A database created before the trigger migration gets FTS triggers added by
// Open(), so a raw INSERT (no application-level rebuild) becomes searchable.
func TestMigrate_AddsFTSTriggersToOldDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old_fts.db")
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
	remote       TEXT NOT NULL DEFAULT '',
	branch       TEXT NOT NULL DEFAULT '',
	tags         TEXT NOT NULL DEFAULT '',
	status       TEXT NOT NULL DEFAULT 'active',
	device       TEXT NOT NULL DEFAULT '',
	created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE migrations (name TEXT PRIMARY KEY);
CREATE VIRTUAL TABLE parks_fts USING fts5(
	name, description, body, why, how_to_apply, tags,
	content='parks', content_rowid='id',
	tokenize='porter unicode61'
);
INSERT INTO migrations VALUES('normalize_ssh_remotes');
INSERT INTO migrations VALUES('fts5_init');
`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	conn.Close()

	conn, err = Open(path)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	defer conn.Close()

	// A raw INSERT, bypassing any application-level FTS rebuild, must be
	// indexed by the triggers alone.
	if _, err := conn.Exec(`INSERT INTO parks (name) VALUES ('triggertest')`); err != nil {
		t.Fatalf("insert: %v", err)
	}

	var got string
	if err := conn.QueryRow(`SELECT name FROM parks_fts WHERE parks_fts MATCH 'triggertest'`).Scan(&got); err != nil {
		t.Fatalf("search after raw insert: %v", err)
	}
	if got != "triggertest" {
		t.Errorf("name = %q, want triggertest", got)
	}
}

// A database created before the uid column gets one backfilled per row, and
// the column plus its unique index exist afterwards.
func TestMigrate_BackfillsUIDColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old_uid.db")
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
	remote       TEXT NOT NULL DEFAULT '',
	branch       TEXT NOT NULL DEFAULT '',
	tags         TEXT NOT NULL DEFAULT '',
	status       TEXT NOT NULL DEFAULT 'active',
	device       TEXT NOT NULL DEFAULT '',
	created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE migrations (name TEXT PRIMARY KEY);
INSERT INTO migrations VALUES('normalize_ssh_remotes');
INSERT INTO migrations VALUES('fts5_init');
INSERT INTO migrations VALUES('fts5_triggers');
INSERT INTO parks (name) VALUES ('a');
INSERT INTO parks (name) VALUES ('b');
`); err != nil {
		t.Fatalf("seed: %v", err)
	}
	conn.Close()

	conn, err = Open(path)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	defer conn.Close()

	rows, err := conn.Query(`SELECT uid FROM parks ORDER BY id`)
	if err != nil {
		t.Fatalf("select uid: %v", err)
	}
	defer rows.Close()
	var uids []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if uid == "" {
			t.Errorf("expected backfilled uid, got empty string")
		}
		uids = append(uids, uid)
	}
	if len(uids) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(uids))
	}
	if uids[0] == uids[1] {
		t.Errorf("expected distinct backfilled uids, got %q twice", uids[0])
	}
}

// The sync_state table tracks per-log-file read offsets for the fold path.
func TestMigrate_CreatesSyncStateTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.db")
	conn, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Exec(`INSERT INTO sync_state (file, offset) VALUES ('host.jsonl', 42)`); err != nil {
		t.Fatalf("insert into sync_state: %v", err)
	}
	var offset int64
	if err := conn.QueryRow(`SELECT offset FROM sync_state WHERE file = 'host.jsonl'`).Scan(&offset); err != nil {
		t.Fatalf("select: %v", err)
	}
	if offset != 42 {
		t.Errorf("offset = %d, want 42", offset)
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
