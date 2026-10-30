package cmd

import (
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
