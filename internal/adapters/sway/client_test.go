package sway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"
)

type fakeServer struct {
	ln   net.Listener
	addr string
	done chan struct{}
}

func newFakeServer(t *testing.T, handle func(conn net.Conn)) *fakeServer {
	t.Helper()
	dir := t.TempDir()
	addr := filepath.Join(dir, "sway.ipc")
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "unix", addr)
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeServer{ln: ln, addr: addr, done: make(chan struct{})}
	go func() {
		defer close(s.done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		handle(conn)
	}()
	return s
}

func (s *fakeServer) close() {
	_ = s.ln.Close()
	<-s.done
}

func TestClientRequestReply(t *testing.T) {
	s := newFakeServer(t, func(conn net.Conn) {
		c := NewConn(conn)
		for {
			m, err := c.ReadMessage()
			if err != nil {
				return
			}
			switch m.Type {
			case RunCommand:
				_ = c.WriteMessage(Message{Type: RunCommand, Payload: []byte("success")})
			case GetVersion:
				_ = c.WriteMessage(Message{Type: GetVersion, Payload: []byte(`{"human_readable":"sway 1.8","build":"x","build_date":"d","build_commit":"c","config_path":"/etc/sway/config"}`)})
			case GetWorkspaces:
				_ = c.WriteMessage(Message{Type: GetWorkspaces, Payload: []byte(`[{"num":1,"name":"1","focused":true}]`)})
			default:
				_ = c.WriteMessage(Message{Type: m.Type, Payload: []byte("ok")})
			}
		}
	})
	defer s.close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cl, err := Dial(ctx, s.addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cl.Close() }()

	m, err := cl.RunCommand(ctx, "exec foot")
	if err != nil {
		t.Fatal(err)
	}
	if string(m.Payload) != "success" {
		t.Fatalf("payload %q", m.Payload)
	}

	v, err := cl.GetVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v.HumanReadable != "sway 1.8" {
		t.Fatalf("version %+v", v)
	}

	ws, err := cl.GetWorkspaces(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 1 || ws[0].Num != 1 || ws[0].Name != "1" || !ws[0].Focused {
		t.Fatalf("workspaces %+v", ws)
	}
}

func TestClientGetOutputsTreeInputs(t *testing.T) {
	// Test that we can send requests with the correct message types
	// and receive responses. The fake server uses the same message types
	// as the client (internal constants), which is fine for this test
	// since we're testing the client's Request logic, not the wire protocol.
	s := newFakeServer(t, func(conn net.Conn) {
		c := NewConn(conn)
		for {
			m, err := c.ReadMessage()
			if err != nil {
				return
			}
			payload, _ := json.Marshal(map[string]string{"type": fmt.Sprintf("%d", m.Type)})
			_ = c.WriteMessage(Message{Type: m.Type, Payload: payload})
		}
	})
	defer s.close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cl, err := Dial(ctx, s.addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cl.Close() }()

	for _, typ := range []MessageType{GetOutputs, GetTree, GetInputs} {
		m, err := cl.Request(ctx, typ, nil)
		if err != nil {
			t.Fatalf("Request %v: %v", typ, err)
		}
		var got map[string]string
		if err := json.Unmarshal(m.Payload, &got); err != nil {
			t.Fatal(err)
		}
		if got["type"] != fmt.Sprintf("%d", typ) {
			t.Fatalf("type mismatch for %v", typ)
		}
	}
}

func TestClientSubscribe(t *testing.T) {
	// Test that Subscribe works and we receive events with correct types.
	// The fake server sends events with internal event type constants.
	s := newFakeServer(t, func(conn net.Conn) {
		c := NewConn(conn)
		m, err := c.ReadMessage()
		if err != nil || m.Type != Subscribe {
			return
		}
		_ = c.WriteMessage(Message{Type: Subscribe, Payload: []byte(`{"success":true}`)})
		_ = c.WriteMessage(Message{Type: EventWindow, Payload: []byte(`{"change":"new"}`)})
		_ = c.WriteMessage(Message{Type: EventWorkspace, Payload: []byte(`{"change":"focus"}`)})
	})
	defer s.close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cl, err := Dial(ctx, s.addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cl.Close() }()

	ch, err := cl.Subscribe(ctx, []string{"window", "workspace"})
	if err != nil {
		t.Fatal(err)
	}
	got := []string{}
	for i := 0; i < 2; i++ {
		select {
		case m := <-ch:
			got = append(got, string(m.Payload))
		case <-time.After(3 * time.Second):
			t.Fatal("timed out waiting for event")
		}
	}
	if len(got) != 2 {
		t.Fatalf("got %d events", len(got))
	}
}

// TestClientConnectionLoss tests that a dropped connection returns an error.
func TestClientConnectionLoss(t *testing.T) {
	s := newFakeServer(t, func(conn net.Conn) {
		c := NewConn(conn)
		// Read one message, reply, then close the connection (simulating server crash after reply)
		m, err := c.ReadMessage()
		if err != nil {
			return
		}
		// Reply to first request
		_ = c.WriteMessage(Message{Type: m.Type, Payload: []byte(`{"human_readable":"sway 1.8"}`)})
		// Now close connection without reading more
	})
	defer s.close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cl, err := Dial(ctx, s.addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cl.Close() }()

	// First request should work (server reads it and replies)
	_, err = cl.Request(ctx, GetVersion, nil)
	if err != nil {
		t.Fatalf("first request: %v", err)
	}

	// Second request should fail because connection is closed
	_, err = cl.Request(ctx, GetVersion, nil)
	if err == nil {
		t.Fatal("expected error on second request after connection loss")
	}
	t.Logf("got expected error: %v", err)
}

// TestClientReaderDone tests that a Request issued after the reader died
// returns an error immediately (doesn't wait for context).
func TestClientReaderDone(t *testing.T) {
	s := newFakeServer(t, func(conn net.Conn) {
		c := NewConn(conn)
		// Read one message then close the connection (simulating server crash)
		_, err := c.ReadMessage()
		if err != nil {
			return
		}
		// Reply to first request
		_ = c.WriteMessage(Message{Type: GetVersion, Payload: []byte(`{"human_readable":"sway 1.8"}`)})
		// Close connection
	})
	defer s.close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cl, err := Dial(ctx, s.addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cl.Close() }()

	// First request should work (server reads it and replies)
	_, err = cl.Request(ctx, GetVersion, nil)
	if err != nil {
		t.Fatalf("first request: %v", err)
	}

	// Give time for server to close connection and reader to die
	time.Sleep(100 * time.Millisecond)

	// Second request should fail immediately with "reader stopped" error
	// (not wait for context timeout)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()
	start := time.Now()
	_, err = cl.Request(ctx2, GetVersion, nil)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected error on request after reader died")
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("request took %v, should have returned immediately", elapsed)
	}
	if !errors.Is(err, errors.New("sway: reader stopped")) && err.Error() != "sway: reader stopped" {
		t.Logf("got error: %v", err)
	}
}

// TestClientConcurrentRequests tests that two concurrent GET_* requests both get answered.
func TestClientConcurrentRequests(t *testing.T) {
	s := newFakeServer(t, func(conn net.Conn) {
		c := NewConn(conn)
		for {
			m, err := c.ReadMessage()
			if err != nil {
				return
			}
			// Reply to both request types
			switch m.Type {
			case GetWorkspaces:
				_ = c.WriteMessage(Message{Type: GetWorkspaces, Payload: []byte(`[{"num":1,"name":"1"}]`)})
			case GetOutputs:
				_ = c.WriteMessage(Message{Type: GetOutputs, Payload: []byte(`[{"id":1,"name":"eDP-1"}]`)})
			}
		}
	})
	defer s.close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cl, err := Dial(ctx, s.addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cl.Close() }()

	// Fire two concurrent requests
	done := make(chan error, 2)
	go func() {
		_, err := cl.Request(ctx, GetWorkspaces, nil)
		done <- err
	}()
	go func() {
		_, err := cl.Request(ctx, GetOutputs, nil)
		done <- err
	}()

	for i := 0; i < 2; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("concurrent request failed: %v", err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("timed out waiting for concurrent requests")
		}
	}
}

// TestClientFailingCommand tests that a command with success:false returns an error.
func TestClientFailingCommand(t *testing.T) {
	s := newFakeServer(t, func(conn net.Conn) {
		c := NewConn(conn)
		for {
			m, err := c.ReadMessage()
			if err != nil {
				return
			}
			if m.Type == RunCommand {
				// Reply with success:false
				_ = c.WriteMessage(Message{Type: RunCommand, Payload: []byte(`[{"success":false,"error":"command failed"}]`)})
			}
		}
	})
	defer s.close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cl, err := Dial(ctx, s.addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cl.Close() }()

	_, err = cl.RunCommand(ctx, "invalid command")
	if err == nil {
		t.Fatal("expected error for failing command")
	}
	if !errors.Is(err, errors.New("command failed")) && err.Error() != "command failed" {
		t.Logf("got error: %v", err)
	}
}

// TestClientEventChannelClosesOnDisconnect tests that event channels are closed when the connection drops.
func TestClientEventChannelClosesOnDisconnect(t *testing.T) {
	s := newFakeServer(t, func(conn net.Conn) {
		c := NewConn(conn)
		m, err := c.ReadMessage()
		if err != nil || m.Type != Subscribe {
			return
		}
		_ = c.WriteMessage(Message{Type: Subscribe, Payload: []byte(`{"success":true}`)})
		// Send one event
		_ = c.WriteMessage(Message{Type: EventWindow, Payload: []byte(`{"change":"new"}`)})
		// Then close connection without sending more
	})
	defer s.close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cl, err := Dial(ctx, s.addr)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cl.Close() }()

	ch, err := cl.Subscribe(ctx, []string{"window"})
	if err != nil {
		t.Fatal(err)
	}

	// Should receive the one event
	select {
	case m := <-ch:
		if string(m.Payload) != `{"change":"new"}` {
			t.Fatalf("unexpected payload: %q", m.Payload)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event")
	}

	// Channel should be closed after connection drops
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("channel should be closed after disconnect")
		}
		t.Log("channel correctly closed after disconnect")
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for channel close")
	}
}
