package synclog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
)

// Event is one entry in a device's append-only log.
type Event struct {
	UID    string         `json:"uid"`
	Op     string         `json:"op"` // "add" | "edit" | "status" | "delete"
	TS     string         `json:"ts"` // RFC3339Nano, UTC
	Device string         `json:"device"`
	Fields map[string]any `json:"fields,omitempty"`
}

var unsafeFilenameChar = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// LogPath returns the sanitised path of a device's log file within dir, so
// callers can locate it (e.g. to check whether it's already seeded) without
// duplicating the sanitisation rule.
func LogPath(dir, device string) string {
	return filepath.Join(dir, unsafeFilenameChar.ReplaceAllString(device, "-")+".jsonl")
}

// Append writes ev as one JSON line to dir/<device>.jsonl, creating the
// directory and file as needed. A single Write of the whole line keeps
// concurrent appenders on the same machine from interleaving partial lines.
func Append(dir, device string, ev Event) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(LogPath(dir, device), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	line, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	line = append(line, '\n')
	if _, err := f.Write(line); err != nil {
		return err
	}
	return f.Sync()
}

// FileSink is the park.Sink implementation for a single machine's log
// directory. It satisfies park.Sink structurally, so this package need not
// import internal/park.
type FileSink struct {
	Dir    string
	Device string
}

func (fs FileSink) Emit(ev Event) error {
	ev.Device = fs.Device
	return Append(fs.Dir, fs.Device, ev)
}
