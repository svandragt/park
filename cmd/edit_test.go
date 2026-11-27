package cmd

import (
	"reflect"
	"strings"
	"testing"

	"github.com/svandragt/park/internal/park"
)

func TestRunEdit_SetsStatus(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "foo"})

	if err := RunEdit(s, []string{"-", "--status", "resolved"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := s.Get(id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != "resolved" {
		t.Errorf("status = %q, want %q", got.Status, "resolved")
	}
}

func TestRunEdit_StatusAndFieldsTogether(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "foo"})

	if err := RunEdit(s, []string{"-", "--desc", "new desc", "--status", "archived"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := s.Get(id)
	if got.Status != "archived" {
		t.Errorf("status = %q, want archived", got.Status)
	}
	if got.Description != "new desc" {
		t.Errorf("description = %q, want %q", got.Description, "new desc")
	}
}

func TestRunEdit_RejectsInvalidStatus(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "foo"})

	err := RunEdit(s, []string{"-", "--status", "bogus"})
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
	if !strings.Contains(err.Error(), "status") {
		t.Errorf("error should mention status: %v", err)
	}

	got, _ := s.Get(id)
	if got.Status != "active" {
		t.Errorf("status changed to %q despite invalid input", got.Status)
	}
}

func TestRunEdit_AppendBodyKeepsExisting(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "foo", Body: "original", HowToApply: ""})

	if err := RunEdit(s, []string{"-", "--append-body", "more", "--append-how", "step"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := s.Get(id)
	if got.Body != "original\n\nmore" {
		t.Errorf("body = %q, want %q", got.Body, "original\n\nmore")
	}
	if got.HowToApply != "step" {
		t.Errorf("how = %q, want %q", got.HowToApply, "step")
	}
}

func TestRunEdit_RejectsBodyWithAppendBody(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "foo", Body: "original"})

	if err := RunEdit(s, []string{"-", "--body", "new", "--append-body", "more"}); err == nil {
		t.Fatal("expected error when combining --body and --append-body")
	}

	got, _ := s.Get(id)
	if got.Body != "original" {
		t.Errorf("body changed to %q despite error", got.Body)
	}
}

func TestRunEdit_RemoteNormalisesAndClearsBranch(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "foo", Remote: "https://old", Branch: "dev"})

	if err := RunEdit(s, []string{"-", "--remote", "git@github.com:svandragt/mdex.git"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := s.Get(id)
	if got.Remote != "https://github.com/svandragt/mdex" || got.Branch != "" {
		t.Errorf("remote/branch = %q/%q", got.Remote, got.Branch)
	}
}

func TestRunEdit_EmptyRemoteClearsBoth(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "foo", Remote: "https://old", Branch: "dev"})

	if err := RunEdit(s, []string{"-", "--remote", ""}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := s.Get(id)
	if got.Remote != "" || got.Branch != "" {
		t.Errorf("remote/branch = %q/%q", got.Remote, got.Branch)
	}
}

func TestRunEdit_StatusUsageRows(t *testing.T) {
	for _, tc := range []struct{ status string }{{"archived"}, {"resolved"}, {"active"}} {
		t.Run(tc.status, func(t *testing.T) {
			s := newTestStore(t)
			id, _ := s.Add(park.Item{Name: "foo"})
			if err := RunEdit(s, []string{"-", "--status", tc.status}); err != nil {
				t.Fatal(err)
			}
			want := []string(nil)
			if tc.status != "active" {
				want = usageFor(t, s, tc.status, "cli", id)
			}
			got := usageRows(t, s)
			if !(len(got) == 0 && len(want) == 0) && !reflect.DeepEqual(got, want) {
				t.Errorf("usage rows = %q, want %q", got, want)
			}
		})
	}
}

func TestRunEdit_NoRowWhenStatusUnchanged(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "foo"})
	_ = s.SetStatus(id, "archived")
	if err := RunEdit(s, []string{"-", "--status", "archived"}); err != nil {
		t.Fatal(err)
	}
	if got := usageRows(t, s); len(got) != 0 {
		t.Errorf("usage rows = %q, want none", got)
	}
}

func TestRunReopen_NoUsageRow(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "foo"})
	_ = s.SetStatus(id, "resolved")
	captureStdout(t, func() error { return RunReopen(s, []string{"1"}) })
	if got := usageRows(t, s); len(got) != 0 {
		t.Errorf("usage rows = %q, want none", got)
	}
}
