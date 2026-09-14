package park

import (
	"database/sql"
	"errors"
	"time"

	"github.com/svandragt/park/internal/synclog"
)

var ErrNotFound = errors.New("park item not found")

type Item struct {
	ID          int64
	Name        string
	Description string
	Type        string
	Body        string
	Why         string
	HowToApply  string
	Remote      string
	Branch      string
	Tags        string
	Status      string
	Device      string
	UID         string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Sink receives one event per successful write, best-effort. A nil sink
// (the default) keeps today's single-machine behaviour unchanged.
type Sink interface {
	Emit(ev synclog.Event) error
}

type Store struct {
	db   *sql.DB
	sink Sink
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) DB() *sql.DB { return s.db }

// SetSink wires up the sync log sink. Pass nil to disable emission again.
func (s *Store) SetSink(sink Sink) { s.sink = sink }

// emit is best-effort: a broken sink must never fail the caller's command,
// since the log is reconciled later by the (not-yet-built) fold path.
func (s *Store) emit(ev synclog.Event) {
	if s.sink == nil {
		return
	}
	s.sink.Emit(ev)
}

func (s *Store) Add(item Item) (int64, error) {
	uid := synclog.NewULID()
	res, err := s.db.Exec(`
INSERT INTO parks (name, description, type, body, why, how_to_apply, remote, branch, tags, device, uid)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.Name, item.Description, item.Type, item.Body,
		item.Why, item.HowToApply, item.Remote, item.Branch,
		item.Tags, item.Device, uid,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	s.emit(synclog.Event{
		UID: uid,
		Op:  "add",
		TS:  time.Now().UTC().Format(time.RFC3339Nano),
		Fields: map[string]any{
			"name": item.Name, "description": item.Description, "type": item.Type,
			"body": item.Body, "why": item.Why, "how_to_apply": item.HowToApply,
			"remote": item.Remote, "branch": item.Branch, "tags": item.Tags,
			"status": "active", "device": item.Device,
		},
	})
	return id, nil
}

type ListFilter struct {
	Status string
	Remote string
	Branch string
	Tag    string
	Type   string
}

func (s *Store) List(f ListFilter) ([]Item, error) {
	query := `SELECT id, name, description, type, body, why, how_to_apply, remote, branch, tags, status, device, uid, created_at, updated_at FROM parks WHERE 1=1`
	args := []any{}

	if f.Status != "" {
		query += ` AND status = ?`
		args = append(args, f.Status)
	}
	if f.Remote != "" {
		query += ` AND remote = ?`
		args = append(args, f.Remote)
	}
	if f.Branch != "" {
		query += ` AND branch = ?`
		args = append(args, f.Branch)
	}
	if f.Tag != "" {
		query += ` AND (',' || tags || ',' LIKE ?)`
		args = append(args, "%,"+f.Tag+",%")
	}
	if f.Type != "" {
		query += ` AND type = ?`
		args = append(args, f.Type)
	}
	query += ` ORDER BY updated_at DESC, id DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

func (s *Store) Search(keyword string, f ListFilter) ([]Item, error) {
	query := `
SELECT p.id, p.name, p.description, p.type, p.body, p.why, p.how_to_apply,
       p.remote, p.branch, p.tags, p.status, p.device, p.uid, p.created_at, p.updated_at
FROM parks_fts f
JOIN parks p ON p.id = f.rowid
WHERE parks_fts MATCH ?`
	args := []any{keyword}
	if f.Status != "" {
		query += ` AND p.status = ?`
		args = append(args, f.Status)
	}
	if f.Remote != "" {
		query += ` AND p.remote = ?`
		args = append(args, f.Remote)
	}
	if f.Branch != "" {
		query += ` AND p.branch = ?`
		args = append(args, f.Branch)
	}
	if f.Tag != "" {
		query += ` AND (',' || p.tags || ',' LIKE ?)`
		args = append(args, "%,"+f.Tag+",%")
	}
	if f.Type != "" {
		query += ` AND p.type = ?`
		args = append(args, f.Type)
	}
	query += ` ORDER BY bm25(parks_fts)`
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

func scanRows(rows *sql.Rows) ([]Item, error) {
	var items []Item
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Name, &it.Description, &it.Type, &it.Body,
			&it.Why, &it.HowToApply, &it.Remote, &it.Branch, &it.Tags,
			&it.Status, &it.Device, &it.UID, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (s *Store) Get(id int64) (*Item, error) {
	row := s.db.QueryRow(`SELECT id, name, description, type, body, why, how_to_apply, remote, branch, tags, status, device, uid, created_at, updated_at FROM parks WHERE id = ?`, id)
	var it Item
	if err := row.Scan(&it.ID, &it.Name, &it.Description, &it.Type, &it.Body,
		&it.Why, &it.HowToApply, &it.Remote, &it.Branch, &it.Tags,
		&it.Status, &it.Device, &it.UID, &it.CreatedAt, &it.UpdatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &it, nil
}

type UpdateFields struct {
	Name        *string
	Description *string
	Body        *string
	Why         *string
	HowToApply  *string
	Tags        *string
	Type        *string
}

func (s *Store) Update(id int64, f UpdateFields) error {
	sets := []string{}
	args := []any{}
	fields := map[string]any{}
	if f.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *f.Name)
		fields["name"] = *f.Name
	}
	if f.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *f.Description)
		fields["description"] = *f.Description
	}
	if f.Body != nil {
		sets = append(sets, "body = ?")
		args = append(args, *f.Body)
		fields["body"] = *f.Body
	}
	if f.Why != nil {
		sets = append(sets, "why = ?")
		args = append(args, *f.Why)
		fields["why"] = *f.Why
	}
	if f.HowToApply != nil {
		sets = append(sets, "how_to_apply = ?")
		args = append(args, *f.HowToApply)
		fields["how_to_apply"] = *f.HowToApply
	}
	if f.Tags != nil {
		sets = append(sets, "tags = ?")
		args = append(args, *f.Tags)
		fields["tags"] = *f.Tags
	}
	if f.Type != nil {
		sets = append(sets, "type = ?")
		args = append(args, *f.Type)
		fields["type"] = *f.Type
	}
	if len(sets) == 0 {
		return nil
	}
	args = append(args, id)
	query := "UPDATE parks SET updated_at = CURRENT_TIMESTAMP"
	for _, s := range sets {
		query += ", " + s
	}
	query += " WHERE id = ?"
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	if uid, err := s.uidFor(id); err == nil {
		s.emit(synclog.Event{UID: uid, Op: "edit", TS: time.Now().UTC().Format(time.RFC3339Nano), Fields: fields})
	}
	return nil
}

func (s *Store) UpdateRemote(oldURL, newURL string) (int64, error) {
	uids, err := s.uidsWhere(`remote = ?`, oldURL)
	if err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`UPDATE parks SET remote = ?, updated_at = CURRENT_TIMESTAMP WHERE remote = ?`, newURL, oldURL)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	for _, uid := range uids {
		s.emit(synclog.Event{UID: uid, Op: "edit", TS: time.Now().UTC().Format(time.RFC3339Nano), Fields: map[string]any{"remote": newURL}})
	}
	return n, nil
}

// uidFor looks up a single item's uid so a write's sync event can carry it
// without threading uid through every call site.
func (s *Store) uidFor(id int64) (string, error) {
	var uid string
	err := s.db.QueryRow(`SELECT uid FROM parks WHERE id = ?`, id).Scan(&uid)
	return uid, err
}

// uidsWhere collects uids affected by a bulk write before it runs, since the
// rows (or their old values) may be gone or changed afterwards.
func (s *Store) uidsWhere(where string, args ...any) ([]string, error) {
	rows, err := s.db.Query(`SELECT uid FROM parks WHERE `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var uids []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		uids = append(uids, uid)
	}
	return uids, rows.Err()
}

func (s *Store) GetLast() (*Item, error) {
	items, err := s.List(ListFilter{})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return &items[0], nil
}

func (s *Store) Prune(before time.Time) (int64, error) {
	cutoff := before.UTC().Format("2006-01-02 15:04:05")
	uids, err := s.uidsWhere(`status IN ('resolved','archived') AND updated_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	res, err := s.db.Exec(
		`DELETE FROM parks WHERE status IN ('resolved','archived') AND updated_at < ?`,
		cutoff,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	for _, uid := range uids {
		s.emit(synclog.Event{UID: uid, Op: "delete", TS: time.Now().UTC().Format(time.RFC3339Nano)})
	}
	return n, nil
}

func (s *Store) Delete(id int64) error {
	uid, uidErr := s.uidFor(id)
	res, err := s.db.Exec(`DELETE FROM parks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	if uidErr == nil {
		s.emit(synclog.Event{UID: uid, Op: "delete", TS: time.Now().UTC().Format(time.RFC3339Nano)})
	}
	return nil
}

func (s *Store) SetStatus(id int64, status string) error {
	res, err := s.db.Exec(`UPDATE parks SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, status, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	if uid, err := s.uidFor(id); err == nil {
		s.emit(synclog.Event{UID: uid, Op: "status", TS: time.Now().UTC().Format(time.RFC3339Nano), Fields: map[string]any{"status": status}})
	}
	return nil
}
