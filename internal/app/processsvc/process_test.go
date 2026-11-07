package processsvc

import (
	"context"
	"sync"
	"testing"
	"time"

	"i3_next/internal/api/contracts"
	"i3_next/internal/api/events"
	"i3_next/internal/app/supervisor"
	"i3_next/internal/core/eventbus"
)

func TestProcessServiceStartStop(t *testing.T) {
	sup := supervisor.New(nil)
	svc := New("shell", nil, sup, supervisor.ProcessSpec{Command: "sleep", Args: []string{"30"}},
		eventbus.New(), nil)
	if svc.Name() != "shell" {
		t.Fatalf("name = %q", svc.Name())
	}
	if got := svc.Optional(); !got {
		t.Fatalf("Optional = %v, want true", got)
	}
	if got := svc.DependsOn(); len(got) != 0 {
		t.Fatalf("DependsOn = %v, want empty", got)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := svc.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer svc.Stop(context.Background())

	deadline := time.Now().Add(5 * time.Second)
	var running bool
	for time.Now().Before(deadline) {
		if st := svc.Health(context.Background()).State; st == contracts.ServiceRunning {
			running = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !running {
		t.Fatalf("process did not reach running state; last health: %v", svc.Health(context.Background()))
	}

	if err := svc.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	deadline = time.Now().Add(5 * time.Second)
	var stopped bool
	for time.Now().Before(deadline) {
		if st := svc.Health(context.Background()).State; st == contracts.ServiceStopped {
			stopped = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !stopped {
		t.Fatalf("process did not reach stopped state; last health: %v", svc.Health(context.Background()))
	}
}

func TestProcessServiceCrashRestartLimit(t *testing.T) {
	bus := eventbus.New()
	sup := supervisor.New(nil)
	svc := New("shell", nil, sup, supervisor.ProcessSpec{
		Command: "sh",
		Args:    []string{"-c", "exit 1"},
		Restart: supervisor.RestartPolicy{
			Enabled:        true,
			MaxRestarts:    1,
			Window:         time.Minute,
			InitialBackoff: 50 * time.Millisecond,
			MaxBackoff:     200 * time.Millisecond,
			Multiplier:     2,
		},
	}, bus, nil)

	var mu sync.Mutex
	var got []events.Event
	stopEvents := eventbus.SubscribeFunc[events.Event](bus, events.TopicServiceStateChanged, func(ev events.Event) {
		mu.Lock()
		got = append(got, ev)
		mu.Unlock()
	})
	defer stopEvents()

	ctx, stopCtx := context.WithTimeout(context.Background(), 15*time.Second)
	defer stopCtx()
	if err := svc.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer svc.Stop(context.Background())

	deadline := time.Now().Add(10 * time.Second)
	var failed bool
	for time.Now().Before(deadline) {
		if st := svc.Health(context.Background()).State; st == contracts.ServiceFailed {
			failed = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !failed {
		t.Fatalf("process did not reach failed state; last health: %v", svc.Health(context.Background()))
	}
	mu.Lock()
	var sawFailed bool
	for _, ev := range got {
		if p, ok := ev.Payload.(events.ServiceStateChanged); ok && p.State == string(contracts.ServiceFailed) {
			sawFailed = true
		}
	}
	mu.Unlock()
	if !sawFailed {
		t.Fatalf("expected a ServiceStateChanged failed event, got %d events", len(got))
	}
}
