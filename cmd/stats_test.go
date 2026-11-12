package cmd

import (
	"strings"
	"testing"
	"time"
)

// statsLines returns RunStats's output with each line's whitespace collapsed,
// so the checks do not depend on column widths.
func statsLines(t *testing.T, args ...string) []string {
	t.Helper()
	s := newTestStore(t)
	for _, row := range [][4]string{
		{"3h", "a", "surfaced", "session-hook"},
		{"2h30m", "a", "viewed", "cli"},
		{"2h", "a", "resolved", "cli"},
		{"72h", "b", "surfaced", "search"},
		{"1h", "c", "viewed", "cli"},
	} {
		d, _ := time.ParseDuration(row[0])
		ts := time.Now().Add(-d).UTC().Format("2006-01-02 15:04:05")
		if _, err := s.DB().Exec(`INSERT INTO usage (ts, item_uid, event, source) VALUES (?, ?, ?, ?)`, ts, row[1], row[2], row[3]); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	return collapse(captureStdout(t, func() error { return RunStats(s, args) }))
}

func collapse(out string) []string {
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		lines = append(lines, strings.Join(strings.Fields(l), " "))
	}
	return lines
}

func wantLines(t *testing.T, lines []string, want ...string) {
	t.Helper()
	for _, w := range want {
		found := false
		for _, l := range lines {
			found = found || l == w
		}
		if !found {
			t.Errorf("output has no line %q:\n%s", w, strings.Join(lines, "\n"))
		}
	}
}

func TestRunStats_PrintsCountsSharesAndAttribution(t *testing.T) {
	lines := statsLines(t)

	wantLines(t, lines,
		"surfaced session-hook 1",
		"surfaced search 1",
		"viewed cli 2",
		"resolved cli 1",
		"Surfaced items: 2",
		"later viewed: 1 (50%)",
		"later resolved: 1 (50%)",
		"How items came back",
		"session-hook 1 1 1",
		"file-hook 0 0 0",
		"search 1 0 0",
		"list-tag 0 0 0",
		"direct 0 1 0",
	)
}

func TestRunStats_DaysLimitsTheWindow(t *testing.T) {
	lines := statsLines(t, "--days", "1")

	wantLines(t, lines,
		"Surfaced items: 1",
		"later viewed: 1 (100%)",
		"search 0 0 0",
	)
}

func TestRunStats_EmptyTablePrintsZeros(t *testing.T) {
	out := captureStdout(t, func() error { return RunStats(newTestStore(t), nil) })
	lines := collapse(out)

	wantLines(t, lines,
		"Surfaced items: 0",
		"later viewed: 0 (0%)",
		"later resolved: 0 (0%)",
		"session-hook 0 0 0",
		"direct 0 0 0",
	)
	if strings.Contains(out, "NaN") {
		t.Errorf("output contains NaN:\n%s", out)
	}
}
