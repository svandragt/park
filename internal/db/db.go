package db

import (
	"database/sql"

	"github.com/svandragt/park/internal/synclog"
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
	remote       TEXT NOT NULL DEFAULT '',
	branch       TEXT NOT NULL DEFAULT '',
	tags         TEXT NOT NULL DEFAULT '',
	status       TEXT NOT NULL DEFAULT 'active',
	device       TEXT NOT NULL DEFAULT '',
	uid          TEXT NOT NULL DEFAULT '',
	created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE IF NOT EXISTS migrations (name TEXT PRIMARY KEY);
`); err != nil {
		return err
	}

	// Databases created before the rename still have git_remote; the pragma
	// check makes this idempotent, so it needs no migrations row.
	var old int
	db.QueryRow(`SELECT count(*) FROM pragma_table_info('parks') WHERE name='git_remote'`).Scan(&old)
	if old > 0 {
		if _, err := db.Exec(`ALTER TABLE parks RENAME COLUMN git_remote TO remote`); err != nil {
			return err
		}
	}

	var applied int
	db.QueryRow(`SELECT count(*) FROM migrations WHERE name='normalize_ssh_remotes'`).Scan(&applied)
	if applied == 0 {
		if _, err := db.Exec(`
UPDATE parks
SET remote = 'https://' || REPLACE(REPLACE(SUBSTR(remote, 5), ':', '/'), '.git', '')
WHERE remote LIKE 'git@%'
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

	db.QueryRow(`SELECT count(*) FROM migrations WHERE name='fts5_triggers'`).Scan(&applied)
	if applied == 0 {
		if _, err := db.Exec(`
CREATE TRIGGER IF NOT EXISTS parks_ai AFTER INSERT ON parks BEGIN
	INSERT INTO parks_fts(rowid, name, description, body, why, how_to_apply, tags)
	VALUES (new.id, new.name, new.description, new.body, new.why, new.how_to_apply, new.tags);
END;
CREATE TRIGGER IF NOT EXISTS parks_ad AFTER DELETE ON parks BEGIN
	INSERT INTO parks_fts(parks_fts, rowid, name, description, body, why, how_to_apply, tags)
	VALUES ('delete', old.id, old.name, old.description, old.body, old.why, old.how_to_apply, old.tags);
END;
CREATE TRIGGER IF NOT EXISTS parks_au AFTER UPDATE ON parks BEGIN
	INSERT INTO parks_fts(parks_fts, rowid, name, description, body, why, how_to_apply, tags)
	VALUES ('delete', old.id, old.name, old.description, old.body, old.why, old.how_to_apply, old.tags);
	INSERT INTO parks_fts(rowid, name, description, body, why, how_to_apply, tags)
	VALUES (new.id, new.name, new.description, new.body, new.why, new.how_to_apply, new.tags);
END;
`); err != nil {
			return err
		}
		// Existing databases may have drifted from hand-rolled rebuilds; make
		// sure the index is consistent before the triggers take over.
		if _, err := db.Exec(`INSERT INTO parks_fts(parks_fts) VALUES('rebuild')`); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO migrations VALUES('fts5_triggers')`); err != nil {
			return err
		}
	}

	// Pragma check (mirrors the git_remote rename above) so a database
	// created before the sync log existed gets the column without erroring
	// on ALTER TABLE ADD COLUMN when it's already there.
	var hasUID int
	db.QueryRow(`SELECT count(*) FROM pragma_table_info('parks') WHERE name='uid'`).Scan(&hasUID)
	if hasUID == 0 {
		if _, err := db.Exec(`ALTER TABLE parks ADD COLUMN uid TEXT NOT NULL DEFAULT ''`); err != nil {
			return err
		}
	}

	db.QueryRow(`SELECT count(*) FROM migrations WHERE name='uid_column'`).Scan(&applied)
	if applied == 0 {
		if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS parks_uid ON parks(uid) WHERE uid != ''`); err != nil {
			return err
		}
		rows, err := db.Query(`SELECT id FROM parks WHERE uid = ''`)
		if err != nil {
			return err
		}
		var ids []int64
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		rows.Close()

		tx, err := db.Begin()
		if err != nil {
			return err
		}
		for _, id := range ids {
			if _, err := tx.Exec(`UPDATE parks SET uid = ? WHERE id = ?`, synclog.NewULID(), id); err != nil {
				tx.Rollback()
				return err
			}
		}
		if _, err := tx.Exec(`INSERT INTO migrations VALUES('uid_column')`); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}

	db.QueryRow(`SELECT count(*) FROM migrations WHERE name='sync_state'`).Scan(&applied)
	if applied == 0 {
		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS sync_state (file TEXT PRIMARY KEY, offset INTEGER NOT NULL)`); err != nil {
			return err
		}
		if _, err := db.Exec(`INSERT INTO migrations VALUES('sync_state')`); err != nil {
			return err
		}
	}
	return nil
}
