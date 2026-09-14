package synclog

import (
	"strings"
	"testing"
	"time"
)

func TestNewULID_Shape(t *testing.T) {
	id := NewULID()
	if len(id) != 26 {
		t.Fatalf("length = %d, want 26", len(id))
	}
	const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	for _, c := range id {
		if !strings.ContainsRune(alphabet, c) {
			t.Fatalf("char %q not in Crockford base32 alphabet", c)
		}
	}
}

func TestNewULID_MonotonicAcrossMilliseconds(t *testing.T) {
	a := NewULID()
	time.Sleep(2 * time.Millisecond)
	b := NewULID()
	if !(a < b) {
		t.Fatalf("expected %q < %q", a, b)
	}
}

func TestNewULID_Unique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		id := NewULID()
		if seen[id] {
			t.Fatalf("duplicate ULID: %s", id)
		}
		seen[id] = true
	}
}
