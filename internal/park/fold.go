package park

import (
	"database/sql"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/svandragt/park/internal/synclog"
)

// An item's device says where it was parked; ev.Device only says which
// machine wrote the log line. Folding must keep the former, or every item
// claims to come from whichever machine happened to fold it.
//
// Fold reads every *.jsonl log in dir (including this device's own — that's
// what makes the local cache rebuildable), applies any events not yet seen,
// and advances each file's stored offset. It runs as one transaction so a
// crash cannot advance an offset past events that were never applied.
//
// The whole fold must not emit: applying events through the normal
// Add/Update/SetStatus/Delete paths would re-append everything just read
// back into this device's own log, growing it without bound. Fold applies
// events with direct SQL instead.
func (s *Store) Fold(dir string) (applied int, err error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return 0, err
	}

	type fileEvents struct {
		name      string
		from      int64
		events    []synclog.Event
		newOffset int64
	}
	var perFile []fileEvents
	var all []synclog.Event

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	for _, path := range files {
		name := filepath.Base(path)
		var from int64
		if err := tx.QueryRow(`SELECT offset FROM sync_state WHERE file = ?`, name).Scan(&from); err != nil && err != sql.ErrNoRows {
			return 0, err
		}
		events, newOffset, err := synclog.ReadNew(dir, name, from)
		if err != nil {
			return 0, err
		}
		perFile = append(perFile, fileEvents{name: name, from: from, events: events, newOffset: newOffset})
		all = append(all, events...)
	}

	// An edit/status/delete can reference an item whose add arrived from a
	// different machine's log, so events from all files must be applied in
	// timestamp order rather than file by file.
	sort.SliceStable(all, func(i, j int) bool { return all[i].TS < all[j].TS })

	for _, ev := range all {
		n, err := applyEvent(tx, ev)
		if err != nil {
			return 0, err
		}
		applied += n
	}

	for _, fe := range perFile {
		if _, err := tx.Exec(`
INSERT INTO sync_state (file, offset) VALUES (?, ?)
ON CONFLICT(file) DO UPDATE SET offset = excluded.offset`, fe.name, fe.newOffset); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return applied, nil
}

// ResetLocal deletes every row from parks and every recorded log offset, so
// a subsequent Fold rebuilds the local cache from the shared log alone.
func (s *Store) ResetLocal() (dropped int64, err error) {
	res, err := s.db.Exec(`DELETE FROM parks`)
	if err != nil {
		return 0, err
	}
	dropped, _ = res.RowsAffected()
	if _, err := s.db.Exec(`DELETE FROM sync_state`); err != nil {
		return 0, err
	}
	return dropped, nil
}

// applyEvent applies one event via direct SQL (no sink emission) and
// reports whether it changed anything.
func applyEvent(tx *sql.Tx, ev synclog.Event) (int, error) {
	// updated_at elsewhere in this package (e.g. Prune's cutoff) is always
	// compared in SQLite's own "YYYY-MM-DD HH:MM:SS" text form; an event's
	// RFC3339 ts must be normalised to that form too, or the '-'/'T'
	// difference in the two formats biases the "<" comparison below.
	ts := normalizeTS(ev.TS)

	switch ev.Op {
	case "add":
		res, err := tx.Exec(`
INSERT INTO parks (name, description, type, body, why, how_to_apply, remote, branch, tags, status, device, uid, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(uid) WHERE uid != '' DO NOTHING`,
			str(ev.Fields["name"]), str(ev.Fields["description"]), strOr(ev.Fields["type"], "project"),
			str(ev.Fields["body"]), str(ev.Fields["why"]), str(ev.Fields["how_to_apply"]),
			str(ev.Fields["remote"]), str(ev.Fields["branch"]), str(ev.Fields["tags"]),
			strOr(ev.Fields["status"], "active"), strOr(ev.Fields["device"], ev.Device), ev.UID, ts, ts,
		)
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		return int(n), nil

	case "edit", "status":
		sets := []string{"updated_at = ?"}
		args := []any{ts}
		for field, col := range editableColumns {
			if v, ok := ev.Fields[field]; ok {
				sets = append(sets, col+" = ?")
				args = append(args, str(v))
			}
		}
		if len(sets) == 1 {
			return 0, nil
		}
		args = append(args, ev.UID, ts)
		query := "UPDATE parks SET " + strings.Join(sets, ", ") + " WHERE uid = ? AND updated_at < ?"
		res, err := tx.Exec(query, args...)
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		return int(n), nil

	case "delete":
		res, err := tx.Exec(`DELETE FROM parks WHERE uid = ?`, ev.UID)
		if err != nil {
			return 0, err
		}
		n, _ := res.RowsAffected()
		return int(n), nil

	default:
		return 0, nil
	}
}

// editableColumns maps an event's field names to columns for "edit"/"status".
var editableColumns = map[string]string{
	"name": "name", "description": "description", "type": "type",
	"body": "body", "why": "why", "how_to_apply": "how_to_apply",
	"remote": "remote", "branch": "branch", "tags": "tags", "status": "status",
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func strOr(v any, def string) string {
	if s, ok := v.(string); ok && s != "" {
		return s
	}
	return def
}

// normalizeTS converts an event's RFC3339Nano ts to SQLite's own
// "YYYY-MM-DD HH:MM:SS" text form. If parsing fails the raw string is used
// as a last resort, so a malformed ts can't abort the whole fold.
func normalizeTS(ts string) string {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return ts
	}
	return t.UTC().Format("2006-01-02 15:04:05")
}
