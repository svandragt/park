package park

import "time"

// UsageCount is one row of the usage table, grouped.
type UsageCount struct {
	Event  string
	Source string
	N      int
}

// UsageStats summarises the usage table for a window.
type UsageStats struct {
	Counts []UsageCount // events in the window, per event and source
	// Attributed counts viewed and resolved events per Source, where Source is
	// that of the item's most recent surfaced event in the 2 hours before it,
	// or "direct" when there is none.
	Attributed []UsageCount
	Surfaced   int // distinct items surfaced in the window
	Viewed     int // of those, items viewed after they were first surfaced
	Resolved   int // of those, items resolved after they were first surfaced
}

// RecordUsage writes one usage row per uid in a single transaction. Usage is
// local only: it emits no sync event, and ResetLocal leaves it alone.
func (s *Store) RecordUsage(uids []string, event, source string) error {
	if len(uids) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, uid := range uids {
		if _, err := tx.Exec(`INSERT INTO usage (item_uid, event, source) VALUES (?, ?, ?)`, uid, event, source); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// usageOrder lists the events in the order a reader follows an item.
const usageOrder = `CASE event WHEN 'surfaced' THEN 0 WHEN 'viewed' THEN 1 ELSE 2 END`

// Usage summarises the usage rows since the given time. ts has one-second
// resolution, so events in the same second as the one they are compared with
// count as later and as preceding.
func (s *Store) Usage(since time.Time) (UsageStats, error) {
	var st UsageStats
	cutoff := since.UTC().Format("2006-01-02 15:04:05")

	var err error
	if st.Counts, err = s.usageCounts(`
SELECT event, source, count(*) FROM usage WHERE ts >= ?
GROUP BY event, source ORDER BY `+usageOrder+`, source`, cutoff); err != nil {
		return st, err
	}
	if st.Attributed, err = s.usageCounts(`
SELECT event, via, count(*) FROM (
	SELECT e.event AS event, coalesce((
		SELECT s.source FROM usage s
		WHERE s.item_uid = e.item_uid AND s.event = 'surfaced'
		  AND s.ts <= e.ts AND s.ts >= datetime(e.ts, '-2 hours')
		ORDER BY s.ts DESC, s.rowid DESC LIMIT 1
	), 'direct') AS via
	FROM usage e WHERE e.event IN ('viewed', 'resolved') AND e.ts >= ?
) GROUP BY event, via ORDER BY `+usageOrder+`, via`, cutoff); err != nil {
		return st, err
	}

	err = s.db.QueryRow(`
WITH first AS (
	SELECT item_uid, min(ts) AS ts FROM usage WHERE event = 'surfaced' AND ts >= ? GROUP BY item_uid
)
SELECT count(*),
	coalesce(sum(EXISTS(SELECT 1 FROM usage u WHERE u.item_uid = first.item_uid AND u.event = 'viewed' AND u.ts >= first.ts)), 0),
	coalesce(sum(EXISTS(SELECT 1 FROM usage u WHERE u.item_uid = first.item_uid AND u.event = 'resolved' AND u.ts >= first.ts)), 0)
FROM first`, cutoff).Scan(&st.Surfaced, &st.Viewed, &st.Resolved)
	return st, err
}

func (s *Store) usageCounts(query string, args ...any) ([]UsageCount, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []UsageCount
	for rows.Next() {
		var c UsageCount
		if err := rows.Scan(&c.Event, &c.Source, &c.N); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
