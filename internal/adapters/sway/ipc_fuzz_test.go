package sway

import (
	"io"
	"net"
	"testing"
	"time"
)

type stubConn struct {
	io.Reader
	io.Closer
}

func (s *stubConn) Write(p []byte) (int, error)      { return len(p), nil }
func (s *stubConn) Close() error                     { return s.Closer.Close() }
func (s *stubConn) LocalAddr() net.Addr              { return nil }
func (s *stubConn) RemoteAddr() net.Addr             { return nil }
func (s *stubConn) SetDeadline(time.Time) error      { return nil }
func (s *stubConn) SetReadDeadline(time.Time) error  { return nil }
func (s *stubConn) SetWriteDeadline(time.Time) error { return nil }

type chunkReader struct {
	data  []byte
	off   int
	chunk int
}

func (c *chunkReader) Read(p []byte) (int, error) {
	if c.off >= len(c.data) {
		return 0, io.EOF
	}
	avail := len(c.data) - c.off
	n := c.chunk
	if n > avail {
		n = avail
	}
	if n > len(p) {
		n = len(p)
	}
	n = copy(p, c.data[c.off:c.off+n])
	c.off += n
	return n, nil
}

func TestFuzzFrameSeeds(t *testing.T) {
	// valid minimal message: magic + length 0 + type 0
	valid := append([]byte(magic), make([]byte, headerSize)...)
	rc := &stubConn{Reader: &chunkReader{data: valid, chunk: 3}}
	c := NewConn(rc)
	m, err := c.ReadMessage()
	if err != nil {
		t.Fatalf("valid seed: %v", err)
	}
	if m.Type != 0 {
		t.Fatalf("type %v", m.Type)
	}

	// bad magic
	bad := append([]byte("BADMAG"), make([]byte, headerSize)...)
	rc = &stubConn{Reader: &chunkReader{data: bad, chunk: 3}}
	c = NewConn(rc)
	if _, err := c.ReadMessage(); err == nil {
		t.Fatal("expected bad magic error")
	}

	// huge length header claims maxPayload+1 but no payload follows
	huge := append([]byte(magic), []byte{0xff, 0xff, 0xff, 0xff}...)
	huge = append(huge, make([]byte, 4)...)
	rc = &stubConn{Reader: &chunkReader{data: huge, chunk: 3}}
	c = NewConn(rc)
	if _, err := c.ReadMessage(); err == nil {
		t.Fatal("expected oversized payload error")
	}
}

func FuzzFrame(f *testing.F) {
	f.Add([]byte(magic))
	f.Add([]byte("BADMAG"))
	f.Add(make([]byte, 14))
	f.Fuzz(func(t *testing.T, data []byte) {
		rc := &stubConn{Reader: &chunkReader{data: data, chunk: 7}}
		c := NewConn(rc)
		for i := 0; i < 1000; i++ {
			_, err := c.ReadMessage()
			if err != nil {
				if err == io.EOF {
					return
				}
				return
			}
		}
	})
}
