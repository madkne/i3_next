package sway

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestReconnectingAdapterNameAndInitialState(t *testing.T) {
	a := NewReconnectingAdapter(nil)
	if a.Name() != "sway" {
		t.Fatalf("name = %q, want %q", a.Name(), "sway")
	}
	if a.Connected() {
		t.Fatal("expected not connected initially")
	}
}

func TestReconnectingAdapterStartStopFailingDial(t *testing.T) {
	a := NewReconnectingAdapter(nil)
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
	a := NewReconnectingAdapter(nil)
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
