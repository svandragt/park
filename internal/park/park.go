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
	return res.LastInsertId()
}

func (s *Store) List(status string, remote string) ([]Item, error) {
	query := `SELECT id, name, description, type, body, why, how_to_apply, git_remote, branch, tags, status, device, created_at, updated_at FROM parks WHERE 1=1`
	args := []any{}

	if status != "" {
		query += ` AND status = ?`
		args = append(args, status)
	}
	if remote != "" {
		query += ` AND git_remote = ?`
		args = append(args, remote)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

func (s *Store) UpdateRemote(oldURL, newURL string) (int64, error) {
	res, err := s.db.Exec(`UPDATE parks SET git_remote = ?, updated_at = CURRENT_TIMESTAMP WHERE git_remote = ?`, newURL, oldURL)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
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
	return nil
}
