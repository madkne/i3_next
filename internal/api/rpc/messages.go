// Package rpc implements the JSON-RPC 2.0 control protocol between the
// i3_next daemon and its clients (the shell, i3-nextctl, future tools).
//
// Framing is length-prefixed JSON over a Unix domain socket: each message is
// preceded by a 4-byte big-endian length. The socket is owned by the session
// owner's UID and peer credentials (SO_PEERCRED) are enforced on the server
// side; see Server.
package rpc

// JSON-RPC 2.0 message types.
type MessageType int

const (
	MessageRequest MessageType = iota
	MessageResponse
)

// Request is a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      *int64     `json:"id,omitempty"`
	Method  string      `json:"method"`
	Params  any        `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int64 `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *RPCError `json:"error,omitempty"`
}

// IsNotification reports whether the request is a notification (no id).
func (r *Request) IsNotification() bool { return r.ID == nil }

// Notification is a fire-and-forget request.
type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// NewRequest builds a request.
func NewRequest(method string, params any, id *int64) Request {
	return Request{JSONRPC: JSONRPCVersion, Method: method, Params: params, ID: id}
}

// NewResponse builds a success response.
func NewResponse(id *int64, result any) Response {
	return Response{JSONRPC: JSONRPCVersion, ID: id, Result: result}
}

// NewErrorResponse builds an error response.
func NewErrorResponse(id *int64, err *RPCError) Response {
	return Response{JSONRPC: JSONRPCVersion, ID: id, Error: err}
}

// NewNotification builds a notification.
func NewNotification(method string, params any) Notification {
	return Notification{JSONRPC: JSONRPCVersion, Method: method, Params: params}
}

// ToRequest converts a notification to a request (for uniform handling).
func (n Notification) ToRequest() Request {
	return Request{JSONRPC: JSONRPCVersion, Method: n.Method, Params: n.Params}
}