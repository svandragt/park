package synclog

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
)

// ReadNew reads dir/file from byte offset from to EOF and returns the
// complete JSON-line events found, plus the offset to resume from next time.
//
// A file syncer can deliver a log mid-append, so the tail read may end in a
// partial line; only bytes up to and including the last '\n' are consumed,
// so newOffset never lands inside an incomplete line. If there is no
// complete line beyond from, it returns no events and newOffset == from.
//
// A line that fails to parse as JSON is skipped rather than treated as an
// error, so one corrupt line can't wedge the fold forever.
func ReadNew(dir, file string, from int64) (events []Event, newOffset int64, err error) {
	f, err := os.Open(filepath.Join(dir, file))
	if err != nil {
		return nil, from, err
	}
	defer f.Close()

	if _, err := f.Seek(from, 0); err != nil {
		return nil, from, err
	}
	data, err := readAll(f)
	if err != nil {
		return nil, from, err
	}

	lastNL := bytes.LastIndexByte(data, '\n')
	if lastNL < 0 {
		return nil, from, nil
	}
	complete := data[:lastNL+1]
	newOffset = from + int64(len(complete))

	for _, line := range bytes.Split(complete, []byte{'\n'}) {
		if len(line) == 0 {
			continue
		}
		var ev Event
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		events = append(events, ev)
	}
	return events, newOffset, nil
}

func readAll(f *os.File) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(f); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
