package synclog

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAppend_WritesParseableLinesInOrder(t *testing.T) {
	dir := t.TempDir()
	ev1 := Event{UID: NewULID(), Op: "add", TS: "2026-01-01T00:00:00Z", Device: "host1", Fields: map[string]any{"name": "first"}}
	ev2 := Event{UID: NewULID(), Op: "add", TS: "2026-01-01T00:00:01Z", Device: "host1", Fields: map[string]any{"name": "second"}}

	if err := Append(dir, "host1", ev1); err != nil {
		t.Fatalf("append 1: %v", err)
	}
	if err := Append(dir, "host1", ev2); err != nil {
		t.Fatalf("append 2: %v", err)
	}

	f, err := os.Open(filepath.Join(dir, "host1.jsonl"))
	if err != nil {
		t.Fatalf("open log: %v", err)
	}
	defer f.Close()

	var lines []Event
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var ev Event
		if err := json.Unmarshal(sc.Bytes(), &ev); err != nil {
			t.Fatalf("unparseable line %q: %v", sc.Text(), err)
		}
		lines = append(lines, ev)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0].UID != ev1.UID || lines[1].UID != ev2.UID {
		t.Errorf("events out of order or wrong: %+v", lines)
	}
}

func TestAppend_SanitisesDeviceNameIntoFilename(t *testing.T) {
	dir := t.TempDir()
	ev := Event{UID: NewULID(), Op: "add", TS: "2026-01-01T00:00:00Z", Device: "sander/x670"}

	if err := Append(dir, "sander/x670", ev); err != nil {
		t.Fatalf("append: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 file in dir (device name must not escape it), got %d: %v", len(entries), entries)
	}
	if entries[0].Name() != "sander-x670.jsonl" {
		t.Errorf("filename = %q, want sander-x670.jsonl", entries[0].Name())
	}
}

func TestFileSink_EmitSetsDeviceAndAppends(t *testing.T) {
	dir := t.TempDir()
	sink := FileSink{Dir: dir, Device: "host1"}

	if err := sink.Emit(Event{UID: NewULID(), Op: "add", TS: "2026-01-01T00:00:00Z"}); err != nil {
		t.Fatalf("emit: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "host1.jsonl"))
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	var ev Event
	if err := json.Unmarshal(data[:len(data)-1], &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ev.Device != "host1" {
		t.Errorf("device = %q, want host1", ev.Device)
	}
}

func TestAppend_CreatesDirIfMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "sync")
	ev := Event{UID: NewULID(), Op: "add", TS: "2026-01-01T00:00:00Z", Device: "host1"}

	if err := Append(dir, "host1", ev); err != nil {
		t.Fatalf("append: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "host1.jsonl")); err != nil {
		t.Errorf("expected log file to exist: %v", err)
	}
}
