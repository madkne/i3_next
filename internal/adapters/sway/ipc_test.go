package sway

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestMarshalMessageRoundTrip(t *testing.T) {
	payload := []byte("hello sway")
	m := Message{Type: GetWorkspaces, Payload: payload}
	data, err := MarshalMessage(m)
	if err != nil {
		t.Fatal(err)
	}
	if string(data[:magicSize]) != magic {
		t.Fatalf("bad magic: %q", data[:magicSize])
	}
	length, typ, err := unmarshalHeader(data[magicSize:frameSize])
	if err != nil {
		t.Fatal(err)
	}
	if length != uint32(len(payload)) { //nolint:gosec // test file permissions
		t.Fatalf("length %d, want %d", length, len(payload))
	}
	if typ != GetWorkspaces {
		t.Fatalf("type %v, want %v", typ, GetWorkspaces)
	}
	if !bytes.Equal(data[frameSize:], payload) {
		t.Fatal("payload mismatch")
	}
}

func TestMarshalMessageNativeEndian(t *testing.T) {
	m := Message{Type: GetVersion, Payload: []byte{1, 2, 3, 4}}
	data, err := MarshalMessage(m)
	if err != nil {
		t.Fatal(err)
	}
	length := binary.NativeEndian.Uint32(data[magicSize : magicSize+4])
	if length != 4 {
		t.Fatalf("length %d", length)
	}
	typ := binary.NativeEndian.Uint32(data[magicSize+4 : frameSize])
	if typ != uint32(GetVersion) {
		t.Fatalf("type %d", typ)
	}
}

func TestMarshalMessageTooLarge(t *testing.T) {
	m := Message{Type: RunCommand, Payload: make([]byte, maxPayload+1)}
	if _, err := MarshalMessage(m); err == nil {
		t.Fatal("expected error for oversized payload")
	}
}

func TestIsEvent(t *testing.T) {
	if IsEvent(GetWorkspaces) {
		t.Fatal("GetWorkspaces should not be an event")
	}
	if !IsEvent(EventWindow) {
		t.Fatal("EventWindow should be an event")
	}
}
