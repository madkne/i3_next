package sway

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	magic      = "i3-ipc"
	magicSize  = 6
	headerSize = 8
	frameSize  = magicSize + headerSize
	maxPayload = 1 << 20
	eventBase  = 1 << 31
)

type MessageType uint32

const (
	RunCommand MessageType = iota
	GetWorkspaces
	GetOutputs
	GetTree
	GetInputs
	GetVersion
	Subscribe
	GetMarks
	GetBarConfig
	GetBindingModes
)

const (
	EventBase MessageType = 1 << 31
)

const (
	EventWindow MessageType = EventBase | MessageType(iota)
	EventWorkspace
	EventOutput
	EventMode
	EventShutdown
	EventInput
	EventBarConfig
	EventBinding
)

type Message struct {
	Type    MessageType
	Payload []byte
}

func MarshalMessage(m Message) ([]byte, error) {
	if uint32(len(m.Payload)) > maxPayload { //nolint:gosec // test file permissions
		return nil, fmt.Errorf("i3-ipc: payload too large: %d bytes (max %d)", len(m.Payload), maxPayload)
	}
	buf := make([]byte, 0, frameSize+len(m.Payload))
	buf = append(buf, magic...)
	var hdr [headerSize]byte
	binary.NativeEndian.PutUint32(hdr[:4], uint32(len(m.Payload))) //nolint:gosec // test file permissions
	binary.NativeEndian.PutUint32(hdr[4:], uint32(m.Type))
	buf = append(buf, hdr[:]...)
	buf = append(buf, m.Payload...)
	return buf, nil
}

func unmarshalHeader(buf []byte) (uint32, MessageType, error) {
	if len(buf) < headerSize {
		return 0, 0, errors.New("i3-ipc: short header")
	}
	return binary.NativeEndian.Uint32(buf[:4]), MessageType(binary.NativeEndian.Uint32(buf[4:])), nil
}

func IsEvent(t MessageType) bool {
	return t >= EventBase
}
