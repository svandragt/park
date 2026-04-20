package park

import (
	"database/sql"
	"errors"
	"time"
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
	GitRemote   string
	Branch      string
	Tags        string
	Status      string
	Device      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) rebuildFTS() error {
	_, err := s.db.Exec(`INSERT INTO parks_fts(parks_fts) VALUES('rebuild')`)
	return err
}

func (s *Store) Add(item Item) (int64, error) {
	res, err := s.db.Exec(`
INSERT INTO parks (name, description, type, body, why, how_to_apply, git_remote, branch, tags, device)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.Name, item.Description, item.Type, item.Body,
		item.Why, item.HowToApply, item.GitRemote, item.Branch,
		item.Tags, item.Device,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	_ = s.rebuildFTS()
	return id, nil
}

type ListFilter struct {
	Status string
	Remote string
	Branch string
	Tag    string
}

func (s *Store) List(f ListFilter) ([]Item, error) {
	query := `SELECT id, name, description, type, body, why, how_to_apply, git_remote, branch, tags, status, device, created_at, updated_at FROM parks WHERE 1=1`
	args := []any{}

	if f.Status != "" {
		query += ` AND status = ?`
		args = append(args, f.Status)
	}
	if f.Remote != "" {
		query += ` AND git_remote = ?`
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
	query += ` ORDER BY created_at DESC, id DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

func (s *Store) Search(keyword string) ([]Item, error) {
	rows, err := s.db.Query(`
SELECT p.id, p.name, p.description, p.type, p.body, p.why, p.how_to_apply,
       p.git_remote, p.branch, p.tags, p.status, p.device, p.created_at, p.updated_at
FROM parks_fts f
JOIN parks p ON p.id = f.rowid
WHERE parks_fts MATCH ?
ORDER BY bm25(parks_fts)`, keyword)
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
			&it.Why, &it.HowToApply, &it.GitRemote, &it.Branch, &it.Tags,
			&it.Status, &it.Device, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (s *Store) Get(id int64) (*Item, error) {
	row := s.db.QueryRow(`SELECT id, name, description, type, body, why, how_to_apply, git_remote, branch, tags, status, device, created_at, updated_at FROM parks WHERE id = ?`, id)
	var it Item
	if err := row.Scan(&it.ID, &it.Name, &it.Description, &it.Type, &it.Body,
		&it.Why, &it.HowToApply, &it.GitRemote, &it.Branch, &it.Tags,
		&it.Status, &it.Device, &it.CreatedAt, &it.UpdatedAt); err != nil {
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
	if f.Name != nil {
		sets = append(sets, "name = ?")
		args = append(args, *f.Name)
	}
	if f.Description != nil {
		sets = append(sets, "description = ?")
		args = append(args, *f.Description)
	}
	if f.Body != nil {
		sets = append(sets, "body = ?")
		args = append(args, *f.Body)
	}
	if f.Why != nil {
		sets = append(sets, "why = ?")
		args = append(args, *f.Why)
	}
	if f.HowToApply != nil {
		sets = append(sets, "how_to_apply = ?")
		args = append(args, *f.HowToApply)
	}
	if f.Tags != nil {
		sets = append(sets, "tags = ?")
		args = append(args, *f.Tags)
	}
	if f.Type != nil {
		sets = append(sets, "type = ?")
		args = append(args, *f.Type)
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
	_ = s.rebuildFTS()
	return nil
}

func (s *Store) UpdateRemote(oldURL, newURL string) (int64, error) {
	res, err := s.db.Exec(`UPDATE parks SET git_remote = ?, updated_at = CURRENT_TIMESTAMP WHERE git_remote = ?`, newURL, oldURL)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
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

func (s *Store) Delete(id int64) error {
	res, err := s.db.Exec(`DELETE FROM parks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	_ = s.rebuildFTS()
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
	_ = s.rebuildFTS()
	return nil
}
