package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/svandragt/park/internal/db"
	"github.com/svandragt/park/internal/park"
)

func TestSnapshot_CopiesRowsWithNoWAL(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "park.db")
	conn, err := db.Open(srcPath)
	if err != nil {
		t.Fatalf("open src: %v", err)
	}
	defer conn.Close()
	store := park.New(conn)
	if _, err := store.Add(park.Item{Name: "snapshot-item"}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	path, err := snapshot(store)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	defer os.Remove(path)

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected snapshot file at %s: %v", path, err)
	}
	if _, err := os.Stat(path + "-wal"); !os.IsNotExist(err) {
		t.Errorf("expected no -wal file beside snapshot, stat err = %v", err)
	}

	snapConn, err := db.Open(path)
	if err != nil {
		t.Fatalf("open snapshot: %v", err)
	}
	defer snapConn.Close()
	snapStore := park.New(snapConn)

	items, err := snapStore.List(park.ListFilter{})
	if err != nil {
		t.Fatalf("list snapshot: %v", err)
	}
	if len(items) != 1 || items[0].Name != "snapshot-item" {
		t.Errorf("expected snapshot to contain seeded row, got %+v", items)
	}
}

// serveGet requests path from the web UI mux and returns status and body.
func serveGet(t *testing.T, store *park.Store, path string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	newServeMux(store).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec.Code, rec.Body.String()
}

// seedFamily adds a parent with a resolved and an active child, plus an
// unrelated item.
func seedFamily(t *testing.T, s *park.Store) (parent, done, open *park.Item) {
	t.Helper()
	parent = addParent(t, s, "v0.4")
	add := func(name, uid string) *park.Item {
		id, err := s.Add(park.Item{Name: name, Parent: uid})
		if err != nil {
			t.Fatalf("add %s: %v", name, err)
		}
		it, _ := s.Get(id)
		return it
	}
	done = add("child-done", parent.UID)
	open = add("child-open", parent.UID)
	add("stranger", "")
	if err := s.SetStatus(done.ID, "resolved"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	return parent, done, open
}

func itoa(id int64) string { return strconv.FormatInt(id, 10) }

func TestServeDetail_ParentShowsChildrenWithDoneCount(t *testing.T) {
	s := newTestStore(t)
	parent, done, open := seedFamily(t, s)

	code, body := serveGet(t, s, "/item/"+itoa(parent.ID))
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	for _, want := range []string{
		"Children (1/2 done)",
		`href="/item/` + itoa(done.ID) + `">#` + itoa(done.ID) + ` child-done</a> [resolved]`,
		`href="/item/` + itoa(open.ID) + `">#` + itoa(open.ID) + ` child-open</a> [active]`,
		"parent=" + itoa(parent.ID),
	} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
	if strings.Contains(body, "stranger") {
		t.Error("body lists an unrelated item as a child")
	}
}

func TestServeDetail_ChildLinksToParent(t *testing.T) {
	s := newTestStore(t)
	parent, done, _ := seedFamily(t, s)

	_, body := serveGet(t, s, "/item/"+itoa(done.ID))
	want := `Parent:</span> <a href="/item/` + itoa(parent.ID) + `">#` + itoa(parent.ID) + ` v0.4</a>`
	if !strings.Contains(body, want) {
		t.Errorf("body missing %q", want)
	}
	if strings.Contains(body, "Children (") {
		t.Error("leaf item renders a Children section")
	}
}

func TestServeIndex_ParentFiltersToChildrenAndShowsChip(t *testing.T) {
	s := newTestStore(t)
	parent, _, _ := seedFamily(t, s)

	code, body := serveGet(t, s, "/?status=all&parent="+itoa(parent.ID))
	if code != http.StatusOK {
		t.Fatalf("status = %d", code)
	}
	for _, want := range []string{"child-done", "child-open", "parent: #" + itoa(parent.ID) + " v0.4"} {
		if !strings.Contains(body, want) {
			t.Errorf("body missing %q", want)
		}
	}
	if strings.Contains(body, "stranger") {
		t.Error("body lists a non-child item")
	}
	if !strings.Contains(body, `href="/?status=all">×</a>`) {
		t.Error("chip has no link that drops the parent filter")
	}
}

func TestServeIndex_UnknownParentIs404(t *testing.T) {
	s := newTestStore(t)
	if code, _ := serveGet(t, s, "/?parent=999"); code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
}
