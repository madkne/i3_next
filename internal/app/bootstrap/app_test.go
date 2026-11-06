package bootstrap

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"
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

// TestAcquireLockConcurrency verifies that two concurrent acquirers result in
// exactly one winner, and that the lock file is not removed on release.
func TestAcquireLockConcurrency(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "run")

	app1, err := New(discardLogger(), WithRuntimeDir(dir))
	if err != nil {
		t.Fatalf("New app1: %v", err)
	}
	app2, err := New(discardLogger(), WithRuntimeDir(dir))
	if err != nil {
		t.Fatalf("New app2: %v", err)
	}

	// Hold the first lock until the second has had a chance to contend.
	f1, err1 := app1.acquireLock()
	if err1 != nil {
		t.Fatalf("app1 acquireLock: %v", err1)
	}
	defer func() { _ = f1.Close() }()

	// Second acquirer must fail while the first holds the lock.
	_, err2 := app2.acquireLock()
	if err2 == nil {
		t.Fatal("app2 acquired lock while app1 holds it")
	}
	if !errors.Is(err2, ErrAlreadyRunning) {
		t.Fatalf("app2 err = %v, want %v", err2, ErrAlreadyRunning)
	}

	// Release the first lock.
	_ = syscall.Flock(int(f1.Fd()), syscall.LOCK_UN)
	_ = f1.Close()

	// After release, a new acquirer can take the lock (file not deleted).
	f3, err3 := app2.acquireLock()
	if err3 != nil {
		t.Fatalf("app2 acquireLock after release: %v", err3)
	}
	defer func() { _ = f3.Close() }()

	lockPath := filepath.Join(dir, "i3-next.lock")
	if _, err := os.Stat(lockPath); err != nil {
		t.Fatalf("lock file removed: %v", err)
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
