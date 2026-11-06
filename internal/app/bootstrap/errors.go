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
