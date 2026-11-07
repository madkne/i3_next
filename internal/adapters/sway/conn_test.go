package sway

import (
	"io"
	"net"
	"testing"
)

type fragConn struct {
	net.Conn
	frag int
}

func (f *fragConn) Read(p []byte) (int, error) {
	if cap(p) > f.frag {
		p = p[:f.frag]
	}
	return f.Conn.Read(p)
}

func TestConnReadMessage(t *testing.T) {
	s1, s2 := net.Pipe()
	defer func() { _ = s1.Close() }()
	defer func() { _ = s2.Close() }()
	msg, err := MarshalMessage(Message{Type: GetWorkspaces, Payload: []byte("hello")})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		_, _ = s2.Write(msg)
	}()
	c := NewConn(s1)
	m, err := c.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if m.Type != GetWorkspaces {
		t.Fatalf("type %v", m.Type)
	}
	if string(m.Payload) != "hello" {
		t.Fatalf("payload %q", m.Payload)
	}
}

func TestConnPartialReads(t *testing.T) {
	s1, s2 := net.Pipe()
	defer func() { _ = s1.Close() }()
	defer func() { _ = s2.Close() }()
	full, err := MarshalMessage(Message{Type: GetVersion, Payload: []byte("v1")})
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for i := 0; i < len(full); i++ {
			_, _ = s2.Write(full[i : i+1])
		}
	}()
	c := NewConn(&fragConn{Conn: s1, frag: 1})
	m, err := c.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(m.Payload) != "v1" {
		t.Fatalf("payload %q", m.Payload)
	}
}

func TestConnBadMagic(t *testing.T) {
	s1, s2 := net.Pipe()
	defer func() { _ = s1.Close() }()
	defer func() { _ = s2.Close() }()
	go func() {
		_, _ = s2.Write([]byte("BADMAG\x00\x00\x00\x00\x00\x00\x00\x00"))
	}()
	c := NewConn(s1)
	if _, err := c.ReadMessage(); err == nil {
		t.Fatal("expected bad magic error")
	}
}

func TestConnEOF(t *testing.T) {
	s1, s2 := net.Pipe()
	_ = s2.Close()
	c := NewConn(s1)
	_, err := c.ReadMessage()
	if err != io.EOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

func TestConnUnexpectedEOF(t *testing.T) {
	s1, s2 := net.Pipe()
	defer func() { _ = s1.Close() }()
	defer func() { _ = s2.Close() }()
	go func() {
		_, _ = s2.Write([]byte("i3-ipc\x00\x00\x00\x05"))
		_ = s2.Close()
	}()
	c := NewConn(s1)
	_, err := c.ReadMessage()
	if err != io.ErrUnexpectedEOF {
		t.Fatalf("expected unexpected EOF, got %v", err)
	}
}

func TestConnWriteThenRead(t *testing.T) {
	s1, s2 := net.Pipe()
	defer func() { _ = s1.Close() }()
	defer func() { _ = s2.Close() }()
	c1 := NewConn(s1)
	c2 := NewConn(s2)
	go func() {
		m, _ := c2.ReadMessage()
		_ = c2.WriteMessage(Message{Type: m.Type, Payload: []byte("reply")})
	}()
	if err := c1.WriteMessage(Message{Type: RunCommand, Payload: []byte("cmd")}); err != nil {
		t.Fatal(err)
	}
	m, err := c1.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(m.Payload) != "reply" {
		t.Fatalf("payload %q", m.Payload)
	}
}
