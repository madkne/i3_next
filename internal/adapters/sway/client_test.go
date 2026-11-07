package sway

import (
	"context"
	"encoding/json"
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
	dir := t.TempDir()
	addr := filepath.Join(dir, "sway.ipc")
	ln, err := net.Listen("unix", addr)
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
		defer conn.Close()
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
	defer cl.Close()

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
	defer cl.Close()

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
	defer cl.Close()

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
