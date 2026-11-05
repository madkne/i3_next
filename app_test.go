package bootstrap

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewRequiresLogger(t *testing.T) {
	if _, err := New(nil); !errors.Is(err, ErrLoggerRequired) {
		t.Fatalf("New(nil) error = %v, want %v", err, ErrLoggerRequired)
	}
}

func TestRunStopsOnContextCancel(t *testing.T) {
	app, err := New(discardLogger())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx) }()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}
