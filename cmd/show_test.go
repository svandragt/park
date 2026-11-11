package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/svandragt/park/internal/park"
)

func chdirGitRepo(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	for _, args := range [][]string{
		{"init", "-q"},
		{"remote", "add", "origin", "https://github.com/x/y"},
		{"checkout", "-q", "-b", "feat"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

func TestRunShow_Current(t *testing.T) {
	chdirGitRepo(t)
	s := newTestStore(t)
	remote := "https://github.com/x/y"
	_, _ = s.Add(park.Item{Name: "on-feat", Remote: remote, Branch: "feat"})
	_, _ = s.Add(park.Item{Name: "on-other", Remote: remote, Branch: "other"})

	out := captureStdout(t, func() error { return RunShow(s, []string{"--current"}) })
	if !strings.Contains(out, "on-feat") || strings.Contains(out, "on-other") {
		t.Errorf("unexpected output:\n%s", out)
	}
}

func TestRunShow_CurrentNoMatch(t *testing.T) {
	chdirGitRepo(t)
	s := newTestStore(t)
	_, _ = s.Add(park.Item{Name: "on-other", Remote: "https://github.com/x/y", Branch: "other"})

	err := RunShow(s, []string{"--current"})
	if err == nil || err.Error() != "no active items for this remote and branch" {
		t.Fatalf("got %v", err)
	}
}

// show records a read in last_viewed_at without touching updated_at, which
// would reorder the lists.
func TestRunShow_MarksViewed(t *testing.T) {
	chdirGitRepo(t)
	for name, args := range map[string]func(id int64) []string{
		"id":      func(id int64) []string { return []string{fmt.Sprint(id)} },
		"current": func(int64) []string { return []string{"--current"} },
	} {
		t.Run(name, func(t *testing.T) {
			s := newTestStore(t)
			id, _ := s.Add(park.Item{Name: "seen", Remote: "https://github.com/x/y", Branch: "feat"})
			if _, err := s.DB().Exec(`UPDATE parks SET updated_at = '2020-01-02 03:04:05'`); err != nil {
				t.Fatalf("seed: %v", err)
			}

			captureStdout(t, func() error { return RunShow(s, args(id)) })

			var viewed, updated string
			if err := s.DB().QueryRow(`SELECT last_viewed_at, CAST(updated_at AS TEXT) FROM parks WHERE id = ?`, id).Scan(&viewed, &updated); err != nil {
				t.Fatalf("select: %v", err)
			}
			if viewed == "" {
				t.Error("last_viewed_at not set")
			}
			if updated != "2020-01-02 03:04:05" {
				t.Errorf("updated_at changed to %q", updated)
			}
		})
	}
}
