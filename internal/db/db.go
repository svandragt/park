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
	_, err := db.Exec(`
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
`)
	return err
}
