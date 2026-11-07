package sway

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	coreerrors "i3_next/internal/core/errors"
)

// ErrSocketNotFound indicates SWAYSOCK could not be determined.
var ErrSocketNotFound = fmt.Errorf("%w: sway socket not found", coreerrors.ErrUnavailable)

// SocketPath returns the Sway IPC socket path.
//
// It prefers the SWAYSOCK environment variable. If SWAYSOCK is unset or empty,
// it falls back to running: swaymsg -t get_socketpath.
func SocketPath(ctx context.Context) (string, error) {
	if p := strings.TrimSpace(os.Getenv("SWAYSOCK")); p != "" {
		return p, nil
	}
	return socketPathFromSwaymsg(ctx, "swaymsg")
}

func socketPathFromSwaymsg(ctx context.Context, bin string) (string, error) {
	cmd := exec.CommandContext(ctx, bin, "-t", "get_socketpath")
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			var ee *exec.ExitError
			if errors.As(err, &ee) && len(ee.Stderr) > 0 {
				msg = strings.TrimSpace(string(ee.Stderr))
			}
		}
		if msg != "" {
			return "", fmt.Errorf("%w: %s", ErrSocketNotFound, msg)
		}
		return "", fmt.Errorf("%w: %v", ErrSocketNotFound, err)
	}
	p := strings.TrimSpace(string(out))
	if p == "" {
		return "", fmt.Errorf("%w: swaymsg returned empty socket path", ErrSocketNotFound)
	}
	return p, nil
}
