package bootstrap

import (
	"errors"

	coreerrors "i3_next/internal/core/errors"
)

var ErrLoggerRequired = errors.New("logger is required")

// Merged with internal/core/errors; kept as aliases for compatibility.
var (
	ErrAlreadyRunning = coreerrors.ErrAlreadyRunning
	ErrNotRunning     = coreerrors.ErrNotRunning
)

// Dependency/topological sort errors.
var (
	ErrCycle                = errors.New("dependency cycle")
	ErrDependencyCycle      = ErrCycle
	ErrCircularDependency   = ErrCycle
	ErrUnknownDependency    = errors.New("unknown dependency")
	ErrDuplicateServiceName = errors.New("duplicate service name")
)
