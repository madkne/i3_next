package bootstrap

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"i3_next/internal/core/config"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNewRequiresLogger(t *testing.T) {
	if _, err := New(nil); !errors.Is(err, ErrLoggerRequired) {
		t.Fatalf("New(nil) error = %v, want %v", err, ErrLoggerRequired)
	}
}

func TestBootstrapDirsMode(t *testing.T) {
	base := t.TempDir()
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(base, "run"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(base, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(base, "data"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(base, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(base, "cache"))

	app, err := New(discardLogger())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := app.bootstrapDirs(); err != nil {
		t.Fatalf("bootstrapDirs: %v", err)
	}

	check := func(dir string, want os.FileMode) {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat %s: %v", dir, err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s mode = %o, want %o", dir, got, want)
		}
	}

	check(config.RuntimeDir(), 0o700)
	check(config.ConfigDir(), 0o700)
	check(config.DataDir(), 0o700)
	check(config.StateDir(), 0o700)
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
