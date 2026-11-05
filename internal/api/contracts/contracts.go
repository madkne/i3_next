package contracts

// APIVersion is the version of the internal API contract.
const APIVersion = "v1"

// AppVersion is the application version.
const AppVersion = "0.1.0"

// HealthStatus represents a component health state.
type HealthStatus string

const (
	HealthOK   HealthStatus = "ok"
	HealthWarn HealthStatus = "warn"
	HealthFail HealthStatus = "fail"
)

// ServiceState represents a service lifecycle state.
type ServiceState string

const (
	ServiceStopped  ServiceState = "stopped"
	ServiceStarting ServiceState = "starting"
	ServiceRunning  ServiceState = "running"
	ServiceStopping ServiceState = "stopping"
	ServiceFailed   ServiceState = "failed"
	ServiceDegraded ServiceState = "degraded"
)

// ErrorCode identifies a machine-readable API error.
type ErrorCode string

const (
	ErrCodeNotFound       ErrorCode = "not_found"
	ErrCodeInvalidRequest ErrorCode = "invalid_request"
	ErrCodePermission     ErrorCode = "permission_denied"
	ErrCodeUnavailable    ErrorCode = "unavailable"
	ErrCodeInternal       ErrorCode = "internal"
	ErrCodeTimeout        ErrorCode = "timeout"
)

// APIError is a structured error returned by the API layer.
type APIError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

func (e APIError) Error() string { return e.Message }

// Output describes a display output.
type Output struct {
	Name    string  `json:"name"`
	Width   int     `json:"width"`
	Height  int     `json:"height"`
	Scale   float64 `json:"scale"`
	Primary bool    `json:"primary"`
}

// Workspace describes a workspace.
type Workspace struct {
	Number  int    `json:"number"`
	Name    string `json:"name"`
	Output  string `json:"output"`
	Focused bool   `json:"focused"`
}

// Window describes a top-level window.
type Window struct {
	ID        uint64 `json:"id"`
	Title     string `json:"title"`
	AppID     string `json:"app_id"`
	Workspace int    `json:"workspace"`
	Focused   bool   `json:"focused"`
}

// ServiceInfo describes a registered service.
type ServiceInfo struct {
	Name   string       `json:"name"`
	State  ServiceState `json:"state"`
	Health HealthStatus `json:"health"`
}
