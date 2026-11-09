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
	RunCommand      MessageType = iota // 0
	GetWorkspaces                      // 1
	Subscribe       = 2                // sway uses 2 for Subscribe
	GetOutputs      = 3                // sway uses 3 for GetOutputs
	GetTree         = 4                // sway uses 4 for GetTree
	GetMarks        = 5                // sway uses 5 for GetMarks
	GetBarConfig    = 6                // sway uses 6 for GetBarConfig
	GetVersion      = 7                // sway uses 7 for GetVersion
	GetBindingModes = 8                // sway uses 8 for GetBindingModes
	GetConfig       = 9                // sway uses 9 for GetConfig
	SendTick        = 10               // sway uses 10 for SendTick
	Sync            = 11               // sway uses 11 for Sync
	GetBindingState = 12               // sway uses 12 for GetBindingState
	GetInputs       = 100              // sway uses 100 for GetInputs
	GetSeats        = 101              // sway uses 101 for GetSeats
)

const (
	EventBase MessageType = 1 << 31
)

const (
	EventWorkspace MessageType = EventBase | MessageType(iota) // 0
	EventOutput                                                // 1
	EventMode                                                  // 2
	EventWindow                                                // 3
	_                                                          // 4 (unused)
	_                                                          // 5 (unused)
	EventShutdown                                              // 6
	_                                                          // 7 (unused)
	_                                                          // 8 (unused)
	_                                                          // 9 (unused)
	_                                                          // 10 (unused)
	_                                                          // 11 (unused)
	_                                                          // 12 (unused)
	_                                                          // 13 (unused)
	_                                                          // 14 (unused)
	_                                                          // 15 (unused)
	_                                                          // 16 (unused)
	_                                                          // 17 (unused)
	_                                                          // 18 (unused)
	_                                                          // 19 (unused)
	_                                                          // 20 (unused)
	EventInput                                                 // 21
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
