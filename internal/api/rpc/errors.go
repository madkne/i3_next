// Package rpc implements the JSON-RPC 2.0 control protocol between the
// i3_next daemon and its clients (the shell, i3-nextctl, future tools).
//
// Framing is length-prefixed JSON over a Unix domain socket: each message is
// preceded by a 4-byte big-endian length. The socket is owned by the session
// owner's UID and peer credentials (SO_PEERCRED) are enforced on the server
// side; see Server.
package rpc

import "i3_next/internal/api/contracts"

// JSON-RPC 2.0 protocol constants.
const (
	JSONRPCVersion = "2.0"

	// Standard JSON-RPC 2.0 error codes.
	ErrCodeParseError     int32 = -32700
	ErrCodeInvalidRequest int32 = -32600
	ErrCodeMethodNotFound int32 = -32601
	ErrCodeInvalidParams  int32 = -32602
	ErrCodeInternal      int32 = -32603

	// Server-error range (-32000..-32099) used for i3_next-specific failures.
	ErrCodeUnauthorized      int32 = -32000
	ErrCodeVersionMismatch   int32 = -32001
	ErrCodeMessageTooLarge   int32 = -32002
	ErrCodeTimeout           int32 = -32003
	ErrCodeUnavailable       int32 = -32004
)

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int32 `json:"code"`
	Message string `json:"message"`
	Data    any   `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return e.CodeString()
}

// CodeString returns a human-readable name for the error code.
func (e *RPCError) CodeString() string {
	switch e.Code {
	case ErrCodeParseError:
		return "parse error"
	case ErrCodeInvalidRequest:
		return "invalid request"
	case ErrCodeMethodNotFound:
		return "method not found"
	case ErrCodeInvalidParams:
		return "invalid params"
	case ErrCodeInternal:
		return "internal error"
	case ErrCodeUnauthorized:
		return "unauthorized"
	case ErrCodeVersionMismatch:
		return "version mismatch"
	case ErrCodeMessageTooLarge:
		return "message too large"
	case ErrCodeTimeout:
		return "timeout"
	case ErrCodeUnavailable:
		return "unavailable"
	default:
		return "server error"
	}
}

// Predefined error values.
var (
	ErrParseError      = &RPCError{Code: ErrCodeParseError, Message: "Parse error"}
	ErrInvalidRequest  = &RPCError{Code: ErrCodeInvalidRequest, Message: "Invalid Request"}
	ErrMethodNotFound  = &RPCError{Code: ErrCodeMethodNotFound, Message: "Method not found"}
	ErrInvalidParams   = &RPCError{Code: ErrCodeInvalidParams, Message: "Invalid params"}
	ErrInternal        = &RPCError{Code: ErrCodeInternal, Message: "Internal error"}
	ErrUnauthorized    = &RPCError{Code: ErrCodeUnauthorized, Message: "Unauthorized: peer credentials rejected"}
	ErrVersionMismatch = &RPCError{Code: ErrCodeVersionMismatch, Message: "Version mismatch"}
	ErrMessageTooLarge = &RPCError{Code: ErrCodeMessageTooLarge, Message: "Message too large"}
	ErrTimeout         = &RPCError{Code: ErrCodeTimeout, Message: "Timeout"}
	ErrUnavailable     = &RPCError{Code: ErrCodeUnavailable, Message: "Unavailable"}
)

// ToContract maps an RPC error onto the shared contracts.ErrorCode.
func (e *RPCError) ToContract() contracts.ErrorCode {
	if e == nil {
		return contracts.ErrCodeInternal
	}
	switch e.Code {
	case ErrCodeUnauthorized:
		return contracts.ErrCodePermission
	case ErrCodeVersionMismatch, ErrCodeUnavailable:
		return contracts.ErrCodeUnavailable
	case ErrCodeTimeout:
		return contracts.ErrCodeTimeout
	case ErrCodeInvalidParams:
		return contracts.ErrCodeInvalidRequest
	case ErrCodeMethodNotFound, ErrCodeParseError, ErrCodeInvalidRequest:
		return contracts.ErrCodeInvalidRequest
	default:
		return contracts.ErrCodeInternal
	}
}