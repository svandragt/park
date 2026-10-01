package cmd

import (
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/svandragt/park/internal/park"
	"github.com/svandragt/park/internal/synclog"
)

func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	runErr := fn()
	os.Stdout = old
	w.Close()
	out, _ := io.ReadAll(r)
	if runErr != nil {
		t.Fatalf("unexpected error: %v", runErr)
	}
	return string(out)
}

// addParent seeds a milestone and returns it.
func addParent(t *testing.T, s *park.Store, name string) *park.Item {
	t.Helper()
	id, err := s.Add(park.Item{Name: name, Type: "project"})
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	it, _ := s.Get(id)
	return it
}

func TestRunAdd_StoresParentUIDAndShowPrintsIt(t *testing.T) {
	t.Chdir(t.TempDir()) // no git remote, so add makes no network lookup
	for _, ref := range []string{"ID", "#ID", "-"} {
		t.Run(ref, func(t *testing.T) {
			s := newTestStore(t)
			parent := addParent(t, s, "v0.4")
			arg := strings.Replace(ref, "ID", strconv.FormatInt(parent.ID, 10), 1)

			if err := RunAdd(s, []string{"--name", "child", "--parent", arg}); err != nil {
				t.Fatalf("add: %v", err)
			}
			child, _ := s.GetLast()
			if child.Parent != parent.UID {
				t.Fatalf("parent = %q, want uid %q", child.Parent, parent.UID)
			}

			out := captureStdout(t, func() error { return RunShow(s, []string{"-"}) })
			want := "Parent: #" + strconv.FormatInt(parent.ID, 10) + " v0.4"
			if !strings.Contains(out, want) {
				t.Errorf("show output missing %q:\n%s", want, out)
			}
		})
	}
}

func TestRunAdd_DefaultsTypeToTask(t *testing.T) {
	t.Chdir(t.TempDir())
	s := newTestStore(t)
	if err := RunAdd(s, []string{"--name", "plain"}); err != nil {
		t.Fatalf("add: %v", err)
	}
	it, _ := s.GetLast()
	if it.Type != "task" {
		t.Errorf("type = %q, want task", it.Type)
	}
}

func TestRunEdit_ParentRoundTripsAndClears(t *testing.T) {
	s := newTestStore(t)
	parent := addParent(t, s, "v0.4")
	cid, _ := s.Add(park.Item{Name: "child"})
	cs := strconv.FormatInt(cid, 10)

	if err := RunEdit(s, []string{cs, "--parent", "#" + strconv.FormatInt(parent.ID, 10)}); err != nil {
		t.Fatalf("set parent: %v", err)
	}
	if got, _ := s.Get(cid); got.Parent != parent.UID {
		t.Fatalf("parent = %q, want %q", got.Parent, parent.UID)
	}

	if err := RunEdit(s, []string{cs, "--parent", ""}); err != nil {
		t.Fatalf("clear parent: %v", err)
	}
	if got, _ := s.Get(cid); got.Parent != "" {
		t.Errorf("parent = %q, want cleared", got.Parent)
	}
}

func TestRunEdit_RejectsSelfParent(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "loop"})
	cs := strconv.FormatInt(id, 10)

	if err := RunEdit(s, []string{cs, "--parent", cs}); err == nil {
		t.Fatal("expected error for self-parent")
	}
	if got, _ := s.Get(id); got.Parent != "" {
		t.Errorf("parent set to %q despite error", got.Parent)
	}
}

func TestResolveParent_UnknownIDErrors(t *testing.T) {
	s := newTestStore(t)
	if _, err := resolveParent(s, "999"); err == nil {
		t.Error("expected error for unknown parent")
	}
}

// seedMilestone returns a parent with two active children and one resolved.
func seedMilestone(t *testing.T, s *park.Store) *park.Item {
	t.Helper()
	parent := addParent(t, s, "v0.4")
	s.Add(park.Item{Name: "open one", Parent: parent.UID})
	s.Add(park.Item{Name: "open two", Parent: parent.UID})
	did, _ := s.Add(park.Item{Name: "finished", Parent: parent.UID})
	s.SetStatus(did, "resolved")
	return parent
}

func TestResolvingParentWithOpenChildrenWarns(t *testing.T) {
	cases := map[string]func(s *park.Store, id string) error{
		"done":        func(s *park.Store, id string) error { return RunDone(s, []string{id}) },
		"edit status": func(s *park.Store, id string) error { return RunEdit(s, []string{id, "--status", "resolved"}) },
	}
	for name, run := range cases {
		t.Run(name, func(t *testing.T) {
			s := newTestStore(t)
			parent := seedMilestone(t, s)

			out := captureStdout(t, func() error { return run(s, strconv.FormatInt(parent.ID, 10)) })

			if !strings.Contains(out, "warning: 2 open child item(s):") {
				t.Errorf("missing warning:\n%s", out)
			}
			if !strings.Contains(out, "open one") || !strings.Contains(out, "open two") {
				t.Errorf("warning should list open children:\n%s", out)
			}
			if strings.Contains(out, "finished") {
				t.Errorf("resolved child should not be listed:\n%s", out)
			}
			if got, _ := s.Get(parent.ID); got.Status != "resolved" {
				t.Errorf("status = %q, warning must not block", got.Status)
			}
		})
	}
}

func TestRunDone_NoWarningWithoutOpenChildren(t *testing.T) {
	s := newTestStore(t)
	id, _ := s.Add(park.Item{Name: "solo"})
	out := captureStdout(t, func() error { return RunDone(s, []string{strconv.FormatInt(id, 10)}) })
	if strings.Contains(out, "warning") {
		t.Errorf("unexpected warning:\n%s", out)
	}
}

func TestRunShow_PrintsChildrenWithDoneCount(t *testing.T) {
	s := newTestStore(t)
	parent := seedMilestone(t, s)
	out := captureStdout(t, func() error { return RunShow(s, []string{strconv.FormatInt(parent.ID, 10)}) })
	if !strings.Contains(out, "Children: 1/3 done") {
		t.Errorf("missing children header:\n%s", out)
	}
	if !strings.Contains(out, "open one") || !strings.Contains(out, "finished") {
		t.Errorf("missing child rows:\n%s", out)
	}
}

func TestRunList_ParentFilterPrintsHeaderWithDoneCount(t *testing.T) {
	s := newTestStore(t)
	parent := seedMilestone(t, s)
	s.Add(park.Item{Name: "unrelated"})
	ps := strconv.FormatInt(parent.ID, 10)

	out := captureStdout(t, func() error { return RunList(s, []string{"--parent", ps}) })

	if !strings.HasPrefix(out, "#"+ps+" v0.4  1/3 done\n") {
		t.Errorf("header wrong:\n%s", out)
	}
	if !strings.Contains(out, "open one") || strings.Contains(out, "unrelated") {
		t.Errorf("rows not filtered to children:\n%s", out)
	}
}

func TestRunSearch_ParentFilterPrintsHeader(t *testing.T) {
	s := newTestStore(t)
	parent := addParent(t, s, "v0.4")
	s.Add(park.Item{Name: "widget inside", Parent: parent.UID})
	s.Add(park.Item{Name: "widget outside"})
	ps := strconv.FormatInt(parent.ID, 10)

	out := captureStdout(t, func() error { return RunSearch(s, []string{"--parent", ps, "widget"}) })

	if !strings.HasPrefix(out, "#"+ps+" v0.4  0/1 done\n") {
		t.Errorf("header wrong:\n%s", out)
	}
	if !strings.Contains(out, "widget inside") || strings.Contains(out, "widget outside") {
		t.Errorf("rows not filtered to children:\n%s", out)
	}
}

func readLog(t *testing.T, dir string) []synclog.Event {
	t.Helper()
	evs, _, err := synclog.ReadNew(dir, "host1.jsonl", 0)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	return evs
}

func TestRunDelete_ClearsChildrenParentWithOneEditEventEach(t *testing.T) {
	s := newTestStore(t)
	parent := seedMilestone(t, s)
	dir := t.TempDir()
	s.SetSink(synclog.FileSink{Dir: dir, Device: "host1"})

	if err := RunDelete(s, []string{strconv.FormatInt(parent.ID, 10)}); err != nil {
		t.Fatalf("delete: %v", err)
	}

	items, _ := s.List(park.ListFilter{Status: ""})
	if len(items) != 3 {
		t.Fatalf("expected 3 surviving children, got %d", len(items))
	}
	for _, it := range items {
		if it.Parent != "" {
			t.Errorf("%q still has parent %q", it.Name, it.Parent)
		}
	}
	edits, deletes := 0, 0
	for _, ev := range readLog(t, dir) {
		switch {
		case ev.Op == "edit" && ev.Fields["parent"] == "":
			edits++
		case ev.Op == "delete" && ev.UID == parent.UID:
			deletes++
		}
	}
	if edits != 3 || deletes != 1 {
		t.Errorf("log has %d parent-clearing edits and %d deletes, want 3 and 1", edits, deletes)
	}
}

func TestRunRetype_RewritesRowsEmitsEditsAndRebuildAgrees(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a", "b", "c"} {
		synclog.Append(dir, "host1", synclog.Event{
			UID: synclog.NewULID(), Op: "add", TS: "2026-01-01T00:00:00Z",
			Fields: map[string]any{"name": name, "type": "project"},
		})
	}
	synclog.Append(dir, "host1", synclog.Event{
		UID: synclog.NewULID(), Op: "add", TS: "2026-01-01T00:00:00Z",
		Fields: map[string]any{"name": "d", "type": "bug"},
	})
	a := newTestStore(t)
	if err := RunRebuild(a, dir, []string{"--yes"}); err != nil {
		t.Fatalf("rebuild a: %v", err)
	}
	a.SetSink(synclog.FileSink{Dir: dir, Device: "host1"})

	out := captureStdout(t, func() error { return RunRetype(a, []string{"project", "task"}) })
	if !strings.Contains(out, "3") {
		t.Errorf("output should report 3 items:\n%s", out)
	}
	if tasks, _ := a.List(park.ListFilter{Type: "task"}); len(tasks) != 3 {
		t.Errorf("a has %d tasks, want 3", len(tasks))
	}
	edits := 0
	for _, ev := range readLog(t, dir) {
		if ev.Op == "edit" && ev.Fields["type"] == "task" {
			edits++
		}
	}
	if edits != 3 {
		t.Errorf("log has %d type edits, want 3", edits)
	}

	b := newTestStore(t)
	if err := RunRebuild(b, dir, []string{"--yes"}); err != nil {
		t.Fatalf("rebuild b: %v", err)
	}
	if tasks, _ := b.List(park.ListFilter{Type: "task"}); len(tasks) != 3 {
		t.Errorf("fresh store has %d tasks after rebuild, want 3", len(tasks))
	}
	if bugs, _ := b.List(park.ListFilter{Type: "bug"}); len(bugs) != 1 {
		t.Errorf("fresh store has %d bugs after rebuild, want 1", len(bugs))
	}
}

func TestRunRetype_RequiresTwoArgs(t *testing.T) {
	s := newTestStore(t)
	if err := RunRetype(s, []string{"project"}); err == nil {
		t.Error("expected usage error")
	}
}
