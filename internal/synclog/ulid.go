// Package synclog owns the append-only per-device event log format used to
// sync park items across machines without a shared, concurrently-written file.
package synclog

import (
	"crypto/rand"
	"time"
)

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

// NewULID returns a 26-char Crockford base32 ULID: 48-bit millisecond
// timestamp followed by 80 bits of randomness, so IDs generated later sort
// after earlier ones as plain strings.
func NewULID() string {
	var b [16]byte
	ms := uint64(time.Now().UnixMilli())
	b[0] = byte(ms >> 40)
	b[1] = byte(ms >> 32)
	b[2] = byte(ms >> 24)
	b[3] = byte(ms >> 16)
	b[4] = byte(ms >> 8)
	b[5] = byte(ms)
	if _, err := rand.Read(b[6:]); err != nil {
		// crypto/rand.Read only fails if the OS source is broken, which is
		// not something a fallback can meaningfully recover from here.
		panic(err)
	}
	return encode(b)
}

// encode base32-encodes 16 bytes (128 bits) into 26 Crockford chars (5 bits
// each == 130 bits, so the top 2 bits of the first char are always zero).
func encode(b [16]byte) string {
	var out [26]byte
	var buf [20]byte // 16 bytes right-padded to a multiple of 5 bits' worth
	copy(buf[:16], b[:])
	for i := 0; i < 26; i++ {
		bitPos := i * 5
		bytePos := bitPos / 8
		bitOff := uint(bitPos % 8)
		// Read up to 2 bytes as a 16-bit window so a 5-bit group never
		// crosses beyond what's available.
		var window uint16
		window = uint16(buf[bytePos]) << 8
		if bytePos+1 < len(buf) {
			window |= uint16(buf[bytePos+1])
		}
		val := byte((window >> (16 - 5 - bitOff)) & 0x1F)
		out[i] = crockford[val]
	}
	return string(out[:])
}
