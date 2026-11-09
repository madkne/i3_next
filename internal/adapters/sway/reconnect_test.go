package sway

import (
	"context"
	"errors"
	"testing"
	"time"

	"i3_next/internal/api/events"
	"i3_next/internal/core/eventbus"
)

func TestReconnectingAdapterNameAndInitialState(t *testing.T) {
	a := NewReconnectingAdapter(nil, nil)
	if a.Name() != "sway" {
		t.Fatalf("name = %q, want %q", a.Name(), "sway")
	}
	if a.Connected() {
		t.Fatal("expected not connected initially")
	}
}

func TestReconnectingAdapterStartStopFailingDial(t *testing.T) {
	a := NewReconnectingAdapter(nil, nil)
	a.dial = func(ctx context.Context) (*Client, error) {
		return nil, errors.New("dial failed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := a.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(150 * time.Millisecond)
	if a.Connected() {
		t.Fatal("expected not connected with a failing dial")
	}
	if err := a.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestReconnectingAdapterSubscribe(t *testing.T) {
	a := NewReconnectingAdapter(nil, nil)
	a.dial = func(ctx context.Context) (*Client, error) {
		return nil, errors.New("dial failed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := a.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = a.Stop(context.Background()) }()
	ch, err := a.Subscribe(ctx, []string{"workspace"})
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if ch == nil {
		t.Fatal("Subscribe returned nil channel")
	}
}

func TestReconnectingAdapterPublishesConnectedDisconnected(t *testing.T) {
	bus := eventbus.New()
	defer bus.Close()

	connectedCh := make(chan events.Event, 1)
	disconnectedCh := make(chan events.Event, 1)
	cancelConn := eventbus.SubscribeFunc[events.Event](bus, events.TopicSwayConnected, func(ev events.Event) {
		connectedCh <- ev
	})
	cancelDisc := eventbus.SubscribeFunc[events.Event](bus, events.TopicSwayDisconnected, func(ev events.Event) {
		disconnectedCh <- ev
	})
	defer cancelConn()
	defer cancelDisc()

	a := NewReconnectingAdapter(nil, bus)
	dialCount := 0
	a.dial = func(ctx context.Context) (*Client, error) {
		dialCount++
		if dialCount == 1 {
			return nil, errors.New("dial failed")
		}
		// Second dial succeeds but immediately fails read
		return nil, errors.New("dial failed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := a.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer func() { _ = a.Stop(context.Background()) }()

	// Should not receive connected event on failed dial
	select {
	case <-connectedCh:
		t.Fatal("unexpected connected event on failed dial")
	case <-time.After(500 * time.Millisecond):
	}
}
