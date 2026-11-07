package processsvc

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	"i3_next/internal/api/contracts"
	"i3_next/internal/api/events"
	"i3_next/internal/app/supervisor"
	coreerrors "i3_next/internal/core/errors"
	"i3_next/internal/core/eventbus"
	"i3_next/internal/core/registry"
)

// Service supervises a single long-lived process and exposes it as a
// registry.Service. The process is restarted by the supervisor according to its
// restart policy; state transitions are published as ServiceStateChanged
// events so the daemon can log them and report degraded.
type Service struct {
	name   string
	depend []string
	sup    *supervisor.Supervisor
	spec   supervisor.ProcessSpec
	bus    *eventbus.Bus
	logger *slog.Logger

	mu     sync.Mutex
	proc   *supervisor.Process
	cancel context.CancelFunc
	done   chan struct{} // closed when the watcher exits
	last   contracts.ServiceState
}

// New builds a supervised process service. spec must include Command/Args and
// optionally a RestartPolicy; depend lists service names that must start first.
func New(name string, depend []string, sup *supervisor.Supervisor, spec supervisor.ProcessSpec, bus *eventbus.Bus, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	spec.Supervisor = sup
	return &Service{
		name:   name,
		depend: append([]string(nil), depend...),
		sup:    sup,
		spec:   spec,
		bus:    bus,
		logger: logger,
		last:   contracts.ServiceStopped,
	}
}

func (s *Service) Name() string { return s.name }

func (s *Service) DependsOn() []string {
	out := make([]string, len(s.depend))
	copy(out, s.depend)
	return out
}

// Optional reports that this service is optional: its failure degrades the
// daemon but does not stop it.
func (s *Service) Optional() bool { return true }

// Start launches the supervised process and begins watching its state.
func (s *Service) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return coreerrors.ErrAlreadyRunning
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.done = make(chan struct{})
	s.mu.Unlock()

	proc, err := s.sup.Start(runCtx, s.spec)
	if err != nil {
		cancel()
		s.mu.Lock()
		s.cancel = nil
		close(s.done)
		s.mu.Unlock()
		return err
	}
	if proc == nil {
		cancel()
		s.mu.Lock()
		s.cancel = nil
		close(s.done)
		s.mu.Unlock()
		return fmt.Errorf("supervisor returned nil process for %s", s.name)
	}
	s.mu.Lock()
	s.proc = proc
	s.mu.Unlock()

	s.publish(contracts.ServiceStarting, "")
	done := s.done
	go s.watch(runCtx, done)
	return nil
}

// Stop stops the supervised process and the watcher.
func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	cancel := s.cancel
	proc := s.proc
	s.cancel = nil
	s.proc = nil
	done := s.done
	s.done = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if proc != nil {
		_ = proc.Stop(ctx)
	}
	if done != nil {
		<-done
	}
	s.publish(contracts.ServiceStopped, "")
	return nil
}

// Health returns the current process health.
func (s *Service) Health(context.Context) contracts.ServiceInfo {
	s.mu.Lock()
	proc := s.proc
	s.mu.Unlock()
	if proc == nil {
		return contracts.ServiceInfo{Name: s.name, State: contracts.ServiceStopped, Health: contracts.HealthWarn}
	}
	return proc.Health(context.Background())
}

func (s *Service) watch(ctx context.Context, done chan struct{}) {
	defer close(done)
	s.mu.Lock()
	proc := s.proc
	s.mu.Unlock()
	if proc == nil {
		return
	}
	var last contracts.ServiceState
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-proc.Done():
			s.publish(stateOf(proc), "process supervision ended")
			return
		case <-ticker.C:
			if st := stateOf(proc); st != last {
				s.publish(st, "")
				last = st
			}
		}
	}
}

func (s *Service) publish(st contracts.ServiceState, reason string) {
	s.mu.Lock()
	if st == s.last {
		s.mu.Unlock()
		return
	}
	s.last = st
	s.mu.Unlock()
	if s.bus == nil {
		return
	}
	health := contracts.HealthOK
	switch st {
	case contracts.ServiceFailed:
		health = contracts.HealthFail
	case contracts.ServiceDegraded:
		health = contracts.HealthWarn
	case contracts.ServiceStopped:
		health = contracts.HealthWarn
	}
	ev := events.NewEvent(events.TypeServiceStateChanged, s.name, events.ServiceStateChanged{
		Version: events.Version,
		Name:    s.name,
		State:   string(st),
		Health:  string(health),
		Reason:  reason,
	})
	_ = eventbus.Publish(s.bus, string(events.TypeServiceStateChanged), ev, eventbus.Essential())
	s.logger.Info("service state changed",
		slog.String("service", s.name),
		slog.String("state", string(st)),
		slog.String("health", string(health)),
		slog.String("reason", reason))
}

func stateOf(p *supervisor.Process) contracts.ServiceState {
	if p == nil {
		return contracts.ServiceStopped
	}
	switch p.State() {
	case string(supervisor.StateRunning):
		return contracts.ServiceRunning
	case string(supervisor.StateStarting), string(supervisor.StateRestarting):
		return contracts.ServiceStarting
	case string(supervisor.StateStopping), string(supervisor.StateStopped):
		return contracts.ServiceStopped
	case string(supervisor.StateDegraded):
		return contracts.ServiceDegraded
	case string(supervisor.StateFailed):
		return contracts.ServiceFailed
	}
	return contracts.ServiceStopped
}

// Compile-time interface checks.
var _ registry.Service = (*Service)(nil)
