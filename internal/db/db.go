package db

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		return nil, err
	}
	return db, nil
}

func migrate(db *sql.DB) error {
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS parks (
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
CREATE TABLE IF NOT EXISTS migrations (name TEXT PRIMARY KEY);
`); err != nil {
		return err
	}

	var applied int
	db.QueryRow(`SELECT count(*) FROM migrations WHERE name='normalize_ssh_remotes'`).Scan(&applied)
	if applied == 0 {
		if _, err := db.Exec(`
UPDATE parks
SET git_remote = 'https://' || REPLACE(REPLACE(SUBSTR(git_remote, 5), ':', '/'), '.git', '')
WHERE git_remote LIKE 'git@%'
`); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO migrations VALUES('normalize_ssh_remotes')`); err != nil {
			return err
		}
	}

	db.QueryRow(`SELECT count(*) FROM migrations WHERE name='fts5_init'`).Scan(&applied)
	if applied == 0 {
		if _, err := db.Exec(`DROP TABLE IF EXISTS parks_fts`); err != nil {
			return err
		}
		if _, err := db.Exec(`
CREATE VIRTUAL TABLE parks_fts USING fts5(
	name, description, body, why, how_to_apply, tags,
	content='parks', content_rowid='id',
	tokenize='porter unicode61'
)`); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO parks_fts(parks_fts) VALUES('rebuild')`); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO migrations VALUES('fts5_init')`); err != nil {
			return err
		}
	}
	return nil
}
