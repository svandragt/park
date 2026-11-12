package park_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/svandragt/park/internal/park"
)

// ago formats a timestamp d before now in SQLite's own text form.
func ago(d time.Duration) string {
	return time.Now().Add(-d).UTC().Format("2006-01-02 15:04:05")
}

func seedUsage(t *testing.T, s *park.Store, ts, uid, event, source string) {
	t.Helper()
	if _, err := s.DB().Exec(`INSERT INTO usage (ts, item_uid, event, source) VALUES (?, ?, ?, ?)`, ts, uid, event, source); err != nil {
		t.Fatalf("seed usage: %v", err)
	}
}

func TestRecordUsage_WritesOneRowPerUID(t *testing.T) {
	s := newTestStore(t)
	sink := &fakeSink{}
	s.SetSink(sink)

	if err := s.RecordUsage([]string{"u1", "u2"}, "surfaced", "file-hook"); err != nil {
		t.Fatalf("RecordUsage: %v", err)
	}
	if err := s.RecordUsage(nil, "surfaced", "file-hook"); err != nil {
		t.Fatalf("RecordUsage with no uids: %v", err)
	}

	rows, err := s.DB().Query(`SELECT item_uid, event, source, CAST(ts AS TEXT) FROM usage ORDER BY item_uid`)
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	defer rows.Close()
	var got []string
	for rows.Next() {
		var uid, event, source, ts string
		if err := rows.Scan(&uid, &event, &source, &ts); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if ts == "" {
			t.Errorf("%s: ts is empty", uid)
		}
		got = append(got, uid+" "+event+" "+source)
	}
	if want := []string{"u1 surfaced file-hook", "u2 surfaced file-hook"}; !reflect.DeepEqual(got, want) {
		t.Errorf("rows = %q, want %q", got, want)
	}
	if len(sink.events) != 0 {
		t.Errorf("RecordUsage emitted %d sync events, want 0", len(sink.events))
	}
}

func TestUsage_CountsSharesAndAttribution(t *testing.T) {
	s := newTestStore(t)
	// A: surfaced twice; the later file-hook sighting is what the view and the
	// resolve are attributed to.
	seedUsage(t, s, ago(10*time.Hour), "A", "surfaced", "session-hook")
	seedUsage(t, s, ago(9*time.Hour+50*time.Minute), "A", "surfaced", "file-hook")
	seedUsage(t, s, ago(9*time.Hour+40*time.Minute), "A", "viewed", "cli")
	seedUsage(t, s, ago(9*time.Hour), "A", "resolved", "cli")
	// B: viewed 3 h after the search, which is outside the 2 h attribution window.
	seedUsage(t, s, ago(5*time.Hour), "B", "surfaced", "search")
	seedUsage(t, s, ago(2*time.Hour), "B", "viewed", "cli")
	// C: viewed without ever being surfaced.
	seedUsage(t, s, ago(time.Hour), "C", "viewed", "cli")
	// D: viewed before it was surfaced, so the view is not "later".
	seedUsage(t, s, ago(4*time.Hour), "D", "viewed", "cli")
	seedUsage(t, s, ago(3*time.Hour), "D", "surfaced", "list-tag")
	// Outside the 30-day window: not counted anywhere.
	seedUsage(t, s, ago(40*24*time.Hour), "E", "surfaced", "file-hook")
	seedUsage(t, s, ago(40*24*time.Hour), "A", "viewed", "cli")

	got, err := s.Usage(time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("Usage: %v", err)
	}

	want := park.UsageStats{
		Counts: []park.UsageCount{
			{Event: "surfaced", Source: "file-hook", N: 1},
			{Event: "surfaced", Source: "list-tag", N: 1},
			{Event: "surfaced", Source: "search", N: 1},
			{Event: "surfaced", Source: "session-hook", N: 1},
			{Event: "viewed", Source: "cli", N: 4},
			{Event: "resolved", Source: "cli", N: 1},
		},
		Attributed: []park.UsageCount{
			{Event: "viewed", Source: "direct", N: 3},
			{Event: "viewed", Source: "file-hook", N: 1},
			{Event: "resolved", Source: "file-hook", N: 1},
		},
		Surfaced: 3, // A, B, D
		Viewed:   2, // A, B
		Resolved: 1, // A
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Usage =\n%+v\nwant\n%+v", got, want)
	}
}

func TestUsage_EmptyTable(t *testing.T) {
	s := newTestStore(t)

	got, err := s.Usage(time.Now().AddDate(0, 0, -30))
	if err != nil {
		t.Fatalf("Usage: %v", err)
	}
	if len(got.Counts) != 0 || len(got.Attributed) != 0 || got.Surfaced != 0 || got.Viewed != 0 || got.Resolved != 0 {
		t.Errorf("Usage on an empty table = %+v, want zeros", got)
	}
}
