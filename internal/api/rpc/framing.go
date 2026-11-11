// Length-prefixed JSON framing used by the
// i3_next control protocol over Unix domain sockets.
//
// Each message is a JSON document prefixed by a 4-byte big-endian length.
// The maximum message size is enforced by MaxMessageSize (default 1 MiB).

package rpc

import (
	"encoding/binary"
	"fmt"
	"io"
)

// MaxMessageSize is the default maximum framed message size.
const MaxMessageSize = 1 << 20 // 1 MiB

// HeaderSize is the length of the length prefix.
const HeaderSize = 4

// MaxHeader is the largest allowed length prefix value.
const MaxHeader = MaxMessageSize

// WriteFrame writes a single framed message to w.
func WriteFrame(w io.Writer, b []byte) error {
	if len(b) > MaxHeader {
		return fmt.Errorf("frame: message too large: %d > %d", len(b), MaxHeader)
	}
	var hdr [HeaderSize]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(b)))
	if err := writeFull(w, hdr[:]); err != nil {
		return fmt.Errorf("frame: header: %w", err)
	}
	if err := writeFull(w, b); err != nil {
		return fmt.Errorf("frame: payload: %w", err)
	}
	return nil
}

func writeFull(w io.Writer, b []byte) error {
	for len(b) > 0 {
		n, err := w.Write(b)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrShortWrite
		}
		b = b[n:]
	}
	return nil
}

// ReadFrame reads a single framed message from r.
// If max is provided and non-zero, it overrides MaxHeader.
func ReadFrame(r io.Reader, max int) ([]byte, error) {
	if max <= 0 {
		max = MaxHeader
	}
	var hdr [HeaderSize]byte
	if err := readFull(r, hdr[:]); err != nil {
		return nil, err
	}
	n := binary.BigEndian.Uint32(hdr[:])
	if n > uint32(max) {
		return nil, fmt.Errorf("frame: message too large: %d > %d", n, max)
	}
	if n == 0 {
		return nil, nil
	}
	b := make([]byte, n)
	if err := readFull(r, b); err != nil {
		return nil, err
	}
	return b, nil
}

func readFull(r io.Reader, b []byte) error {
	for len(b) > 0 {
		n, err := r.Read(b)
		if err != nil {
			return err
		}
		if n <= 0 {
			return io.ErrUnexpectedEOF
		}
		b = b[n:]
	}
	return nil
}