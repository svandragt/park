package synclog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadNew_ReturnsEventsFromOffset(t *testing.T) {
	dir := t.TempDir()
	ev1 := Event{UID: "a", Op: "add", TS: "2026-01-01T00:00:00Z", Device: "host1"}
	ev2 := Event{UID: "b", Op: "add", TS: "2026-01-01T00:00:01Z", Device: "host1"}
	if err := Append(dir, "host1", ev1); err != nil {
		t.Fatalf("append 1: %v", err)
	}

	events, offset1, err := ReadNew(dir, "host1.jsonl", 0)
	if err != nil {
		t.Fatalf("read 1: %v", err)
	}
	if len(events) != 1 || events[0].UID != "a" {
		t.Fatalf("expected 1 event 'a', got %+v", events)
	}

	if err := Append(dir, "host1", ev2); err != nil {
		t.Fatalf("append 2: %v", err)
	}
	events, _, err = ReadNew(dir, "host1.jsonl", offset1)
	if err != nil {
		t.Fatalf("read 2: %v", err)
	}
	if len(events) != 1 || events[0].UID != "b" {
		t.Fatalf("expected 1 event 'b', got %+v", events)
	}
}

// A file syncer can deliver a log mid-append, so the tail may be a partial
// line; ReadNew must only consume complete lines and report an offset that
// excludes the partial one.
func TestReadNew_StopsAtLastCompleteLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "host1.jsonl")
	complete := `{"uid":"a","op":"add","ts":"2026-01-01T00:00:00Z","device":"host1"}` + "\n"
	partial := `{"uid":"b","op":"add"`
	if err := os.WriteFile(path, []byte(complete+partial), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	events, offset, err := ReadNew(dir, "host1.jsonl", 0)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(events) != 1 || events[0].UID != "a" {
		t.Fatalf("expected only the complete line, got %+v", events)
	}
	if int(offset) != len(complete) {
		t.Fatalf("offset = %d, want %d (excluding the partial line)", offset, len(complete))
	}

	// Appending the rest of the partial line later should now yield it.
	rest := `,"ts":"2026-01-01T00:00:01Z","device":"host1"}` + "\n"
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("open for append: %v", err)
	}
	if _, err := f.WriteString(rest); err != nil {
		t.Fatalf("write rest: %v", err)
	}
	f.Close()

	events, _, err = ReadNew(dir, "host1.jsonl", offset)
	if err != nil {
		t.Fatalf("read after completion: %v", err)
	}
	if len(events) != 1 || events[0].UID != "b" {
		t.Fatalf("expected the now-complete line 'b', got %+v", events)
	}
}

// A no-op read (no new complete lines) must return offset == from unchanged.
func TestReadNew_NoCompleteLineReturnsSameOffset(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "host1.jsonl")
	if err := os.WriteFile(path, []byte(`{"uid":"a"`), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	events, offset, err := ReadNew(dir, "host1.jsonl", 0)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected no events, got %+v", events)
	}
	if offset != 0 {
		t.Fatalf("offset = %d, want 0", offset)
	}
}

// Corrupt lines must be skipped, not abort the whole read.
func TestReadNew_SkipsCorruptLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "host1.jsonl")
	content := `{"uid":"a","op":"add","ts":"2026-01-01T00:00:00Z","device":"host1"}` + "\n" +
		`not json` + "\n" +
		`{"uid":"c","op":"add","ts":"2026-01-01T00:00:02Z","device":"host1"}` + "\n"
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatalf("write: %v", err)
	}

	events, _, err := ReadNew(dir, "host1.jsonl", 0)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 valid events (corrupt line skipped), got %d: %+v", len(events), events)
	}
	if events[0].UID != "a" || events[1].UID != "c" {
		t.Fatalf("unexpected events: %+v", events)
	}
}
