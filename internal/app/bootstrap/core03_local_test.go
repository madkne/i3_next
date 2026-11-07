package bootstrap

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"i3_next/internal/api/contracts"
	"i3_next/internal/core/registry"
)

type fakeRollbackSvc struct {
	name     string
	required bool
	startErr error

	mu      sync.Mutex
	started bool
	stopped bool
}

func (f *fakeRollbackSvc) Name() string { return f.name }
func (f *fakeRollbackSvc) Start(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started = true
	return f.startErr
}
func (f *fakeRollbackSvc) Stop(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stopped = true
	return nil
}
func (f *fakeRollbackSvc) Health(ctx context.Context) contracts.ServiceInfo {
	return contracts.ServiceInfo{Name: f.name, Health: contracts.HealthOK, State: contracts.ServiceRunning}
}
func (f *fakeRollbackSvc) Required() bool { return f.required }

func TestCORE03RequiredFailureRollsBack(t *testing.T) {
	s1 := &fakeRollbackSvc{name: "s1", required: true}
	s2 := &fakeRollbackSvc{name: "s2", required: true, startErr: errors.New("boom")}
	app, _ := New(discardLogger(), WithService(s1), WithService(s2))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := app.Run(ctx)
	if err == nil {
		t.Fatal("expected error")
	}
	if !s1.stopped {
		t.Fatal("s1 should be stopped during rollback")
	}
	if app.Degraded() {
		t.Fatal("required failure should not set degraded")
	}
}

func TestCORE03OptionalFailureDegradesAndContinues(t *testing.T) {
	s1 := &fakeRollbackSvc{name: "s1", required: true}
	// optional via method
	s2 := &fakeRollbackSvc{name: "s2", required: false, startErr: errors.New("boom")}
	s3 := &fakeRollbackSvc{name: "s3", required: true}
	app, _ := New(discardLogger(), WithService(s1), WithService(s2), WithService(s3))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx) }()
	time.Sleep(50 * time.Millisecond)
	if !app.Degraded() {
		t.Fatal("expected degraded")
	}
	s3.mu.Lock()
	started := s3.started
	s3.mu.Unlock()
	if !started {
		t.Fatal("s3 should have started")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}
}

func TestCORE03OptionalWrapper(t *testing.T) {
	// override startErr by using inner? WrapOptional wraps base; need fail. Use new fake.
	f := &fakeRollbackSvc{name: "opt", required: true, startErr: errors.New("boom")}
	w := registry.WrapOptional(f)
	app, _ := New(discardLogger(), WithService(w))
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- app.Run(ctx) }()
	time.Sleep(50 * time.Millisecond)
	if !app.Degraded() {
		t.Fatal("expected degraded")
	}
	cancel()
	<-done
	_ = io.Discard
	_ = slog.Default()
}
