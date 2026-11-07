package events

import (
	"fmt"
	"time"
)

// SchemaVersion is the version of the event schema.
const SchemaVersion = "v1"

// Version is an alias for SchemaVersion.
const Version = SchemaVersion

// APIVersion is an alias.
const APIVersion = SchemaVersion

// EventType identifies an event kind.
type EventType string

const (
	TypeSessionStarted      EventType = "session.started"
	TypeSwayConnected       EventType = "sway.connected"
	TypeSwayDisconnected    EventType = "sway.disconnected"
	TypeWorkspaceChanged    EventType = "workspace.changed"
	TypeOutputChanged       EventType = "output.changed"
	TypeConfigChanged       EventType = "config.changed"
	TypeServiceStateChanged EventType = "service.state_changed"
	TypeShutdownRequested   EventType = "shutdown.requested"
)

// Backwards/alternative constant names.
const (
	SessionStartedType      = TypeSessionStarted
	SwayConnectedType       = TypeSwayConnected
	SwayDisconnectedType    = TypeSwayDisconnected
	WorkspaceChangedType    = TypeWorkspaceChanged
	OutputChangedType       = TypeOutputChanged
	ConfigChangedType       = TypeConfigChanged
	ServiceStateChangedType = TypeServiceStateChanged
	ShutdownRequestedType   = TypeShutdownRequested
)

// Topic aliases (string).
const (
	TopicSessionStarted      = string(TypeSessionStarted)
	TopicSwayConnected       = string(TypeSwayConnected)
	TopicSwayDisconnected    = string(TypeSwayDisconnected)
	TopicWorkspaceChanged    = string(TypeWorkspaceChanged)
	TopicOutputChanged       = string(TypeOutputChanged)
	TopicConfigChanged       = string(TypeConfigChanged)
	TopicServiceStateChanged = string(TypeServiceStateChanged)
	TopicShutdownRequested   = string(TypeShutdownRequested)
)

// Event is the versioned event envelope.
type Event struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Version   string    `json:"version"`
	Timestamp time.Time `json:"timestamp"`
	Source    string    `json:"source"`
	Payload   any       `json:"payload"`
}

// SessionStarted payload.
type SessionStarted struct {
	Version   string    `json:"version"`
	SessionID string    `json:"session_id"`
	StartedAt time.Time `json:"started_at"`
	Timestamp time.Time `json:"timestamp"`
}

// SwayConnected payload.
type SwayConnected struct {
	Version    string `json:"version"`
	SocketPath string `json:"socket_path"`
	VersionStr string `json:"sway_version"`
}

// SwayDisconnected payload.
type SwayDisconnected struct {
	Version string `json:"version"`
	Reason  string `json:"reason"`
	Error   string `json:"error"`
}

// WorkspaceChanged payload.
type WorkspaceChanged struct {
	Version string `json:"version"`
	Number  int    `json:"number"`
	Name    string `json:"name"`
	Output  string `json:"output"`
	Focused bool   `json:"focused"`
}

// OutputChanged payload.
type OutputChanged struct {
	Version   string  `json:"version"`
	Name      string  `json:"name"`
	Width     int     `json:"width"`
	Height    int     `json:"height"`
	Scale     float64 `json:"scale"`
	Connected bool    `json:"connected"`
}

// ConfigChanged payload.
type ConfigChanged struct {
	Version string   `json:"version"`
	Path    string   `json:"path"`
	Keys    []string `json:"keys"`
}

// ServiceStateChanged payload.
type ServiceStateChanged struct {
	Version string `json:"version"`
	Name    string `json:"name"`
	State   string `json:"state"`
	Health  string `json:"health"`
	Reason  string `json:"reason"`
}

// ShutdownRequested payload.
type ShutdownRequested struct {
	Version string `json:"version"`
	Reason  string `json:"reason"`
	Source  string `json:"source"`
}

// Event type aliases for payload structs (so callers can use either).
type (
	SessionStartedEvent      = SessionStarted
	SwayConnectedEvent       = SwayConnected
	SwayDisconnectedEvent    = SwayDisconnected
	WorkspaceChangedEvent    = WorkspaceChanged
	OutputChangedEvent       = OutputChanged
	ConfigChangedEvent       = ConfigChanged
	ServiceStateChangedEvent = ServiceStateChanged
	ShutdownRequestedEvent   = ShutdownRequested
)

// NewEvent builds a versioned event envelope.
func NewEvent(t any, source string, payload any) Event {
	et := EventType("")
	switch v := t.(type) {
	case EventType:
		et = v
	case string:
		et = EventType(v)
	default:
		et = EventType(fmt.Sprint(v))
	}
	return Event{
		Type:      et,
		Version:   SchemaVersion,
		Timestamp: time.Now(),
		Source:    source,
		Payload:   payload,
	}
}
