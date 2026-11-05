package errors

import "errors"

// Domain errors shared across core, services, and APIs.
var (
	// ErrNotFound indicates a requested entity does not exist.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists indicates an entity already exists.
	ErrAlreadyExists = errors.New("already exists")
	// ErrInvalidArgument indicates invalid input.
	ErrInvalidArgument = errors.New("invalid argument")
	// ErrPermissionDenied indicates insufficient permissions.
	ErrPermissionDenied = errors.New("permission denied")
	// ErrUnavailable indicates a dependency or service is unavailable.
	ErrUnavailable = errors.New("unavailable")
	// ErrTimeout indicates an operation timed out.
	ErrTimeout = errors.New("timeout")
	// ErrCanceled indicates an operation was canceled.
	ErrCanceled = errors.New("canceled")
	// ErrAlreadyRunning indicates the daemon or service is already running.
	ErrAlreadyRunning = errors.New("already running")
	// ErrNotRunning indicates the daemon or service is not running.
	ErrNotRunning = errors.New("not running")
	// ErrInternal indicates an unexpected internal failure.
	ErrInternal = errors.New("internal error")
)
