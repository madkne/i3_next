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

func TestMessageTypeConstants(t *testing.T) {
	// These values are defined by the sway-ipc(7) protocol specification.
	// If these change, the wire protocol compatibility with sway is broken.
	tests := []struct {
		name string
		got  uint32
		want uint32
	}{
		{"RunCommand", uint32(RunCommand), 0},
		{"GetWorkspaces", uint32(GetWorkspaces), 1},
		{"Subscribe", uint32(Subscribe), 2},
		{"GetOutputs", uint32(GetOutputs), 3},
		{"GetTree", uint32(GetTree), 4},
		{"GetMarks", uint32(GetMarks), 5},
		{"GetBarConfig", uint32(GetBarConfig), 6},
		{"GetVersion", uint32(GetVersion), 7},
		{"GetBindingModes", uint32(GetBindingModes), 8},
		{"GetConfig", uint32(GetConfig), 9},
		{"SendTick", uint32(SendTick), 10},
		{"Sync", uint32(Sync), 11},
		{"GetBindingState", uint32(GetBindingState), 12},
		{"GetInputs", uint32(GetInputs), 100},
		{"GetSeats", uint32(GetSeats), 101},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("%s = %d, want %d (per sway-ipc(7))", tc.name, tc.got, tc.want)
			}
		})
	}
}

func TestEventTypeConstants(t *testing.T) {
	// These values are defined by the sway-ipc(7) protocol specification.
	// Event types are OR'd with EventBase (1<<31).
	tests := []struct {
		name string
		got  uint32
		want uint32
	}{
		{"EventWorkspace", uint32(EventWorkspace &^ EventBase), 0},
		{"EventOutput", uint32(EventOutput &^ EventBase), 1},
		{"EventMode", uint32(EventMode &^ EventBase), 2},
		{"EventWindow", uint32(EventWindow &^ EventBase), 3},
		{"EventShutdown", uint32(EventShutdown &^ EventBase), 6},
		{"EventInput", uint32(EventInput &^ EventBase), 21},
		{"EventBarConfig", uint32(EventBarConfig &^ EventBase), 22},
		{"EventBinding", uint32(EventBinding &^ EventBase), 23},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("%s = %d, want %d (per sway-ipc(7))", tc.name, tc.got, tc.want)
			}
		})
	}
}
