package sway

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"i3_next/internal/api/contracts"
	"i3_next/internal/api/events"
	coreerrors "i3_next/internal/core/errors"
	"i3_next/internal/core/eventbus"
	"i3_next/internal/core/registry"
)

const (
	reconnectInitialBackoff = 100 * time.Millisecond
	reconnectMaxBackoff     = 5 * time.Second
	reconnectMultiplier     = 2.0
)

// eventName maps a sway IPC event message type to its event name string.
var eventName = map[MessageType]string{
	EventWindow:    "window",
	EventWorkspace: "workspace",
	EventOutput:    "output",
	EventMode:      "mode",
	EventShutdown:  "shutdown",
	EventInput:     "input",
}

type sub struct {
	events []string
	ch     chan Message
}

// ReconnectingAdapter is a sway.Adapter that transparently reconnects to sway
// when the IPC connection is lost, resubscribes to the registered events and
// keeps forwarding them to stable subscriber channels. It also implements
// registry.Service so it can be registered and supervised like any other
// service.
type ReconnectingAdapter struct {
	logger *slog.Logger
	dial   func(ctx context.Context) (*Client, error)
	bus    *eventbus.Bus // for publishing SwayConnected/SwayDisconnected

	mu     sync.Mutex
	client *Client
	subs   map[uint64]*sub
	nextID uint64
	events []string // union of all subscribed event names

	readerCancel context.CancelFunc
	readerDone   chan struct{}
	runCtx       context.Context // parent for event readers; set in Start

	connected atomic.Bool
	done      chan struct{}
	cancel    context.CancelFunc
}

// NewReconnectingAdapter creates a reconnecting sway adapter that dials the
// default sway IPC socket.
func NewReconnectingAdapter(logger *slog.Logger, bus *eventbus.Bus) *ReconnectingAdapter {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &ReconnectingAdapter{
		logger: logger,
		dial:   defaultDial,
		bus:    bus,
		subs:   make(map[uint64]*sub),
		done:   make(chan struct{}),
	}
}

func defaultDial(ctx context.Context) (*Client, error) {
	path, err := SocketPath(ctx)
	if err != nil {
		return nil, err
	}
	return Dial(ctx, path)
}

// Name returns the service name.
func (a *ReconnectingAdapter) Name() string { return "sway" }

// Connected reports whether the adapter is currently connected to sway.
func (a *ReconnectingAdapter) Connected() bool {
	return a.connected.Load()
}

// Optional reports that sway is optional: the daemon keeps running (degraded)
// while sway is unavailable and reconnects when it returns.
func (a *ReconnectingAdapter) Optional() bool { return true }

// Start begins the reconnect loop.
func (a *ReconnectingAdapter) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithCancel(ctx)
	a.mu.Lock()
	if a.cancel != nil {
		a.mu.Unlock()
		cancel()
		return coreerrors.ErrAlreadyRunning
	}
	a.cancel = cancel
	a.runCtx = runCtx
	a.done = make(chan struct{})
	a.mu.Unlock()

	go a.run(runCtx)
	return nil
}

// Stop stops the reconnect loop and closes the current connection.
func (a *ReconnectingAdapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	cancel := a.cancel
	a.cancel = nil
	a.runCtx = nil
	client := a.client
	a.client = nil
	if a.readerCancel != nil {
		a.readerCancel()
		a.readerCancel = nil
	}
	a.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if client != nil {
		_ = client.Close()
	}
	<-a.done
	return nil
}

// Health reports whether the adapter is currently connected to sway.
func (a *ReconnectingAdapter) Health(context.Context) contracts.ServiceInfo {
	if a.connected.Load() {
		return contracts.ServiceInfo{Name: "sway", State: contracts.ServiceRunning, Health: contracts.HealthOK}
	}
	return contracts.ServiceInfo{Name: "sway", State: contracts.ServiceDegraded, Health: contracts.HealthWarn}
}

// RunCommand executes a sway command on the current connection.
func (a *ReconnectingAdapter) RunCommand(ctx context.Context, command string) (Message, error) {
	c := a.currentClient()
	if c == nil {
		return Message{}, coreerrors.ErrUnavailable
	}
	return c.RunCommand(ctx, command)
}

func (a *ReconnectingAdapter) currentClient() *Client {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.client
}

func (a *ReconnectingAdapter) GetWorkspaces(ctx context.Context) ([]Workspace, error) {
	c := a.currentClient()
	if c == nil {
		return nil, coreerrors.ErrUnavailable
	}
	return c.GetWorkspaces(ctx)
}

func (a *ReconnectingAdapter) GetOutputs(ctx context.Context) ([]Output, error) {
	c := a.currentClient()
	if c == nil {
		return nil, coreerrors.ErrUnavailable
	}
	return c.GetOutputs(ctx)
}

func (a *ReconnectingAdapter) GetTree(ctx context.Context) (*Node, error) {
	c := a.currentClient()
	if c == nil {
		return nil, coreerrors.ErrUnavailable
	}
	return c.GetTree(ctx)
}

func (a *ReconnectingAdapter) GetInputs(ctx context.Context) ([]Input, error) {
	c := a.currentClient()
	if c == nil {
		return nil, coreerrors.ErrUnavailable
	}
	return c.GetInputs(ctx)
}

func (a *ReconnectingAdapter) GetVersion(ctx context.Context) (*Version, error) {
	c := a.currentClient()
	if c == nil {
		return nil, coreerrors.ErrUnavailable
	}
	return c.GetVersion(ctx)
}

// Subscribe registers a channel for the given sway events. The channel is
// stable across reconnects: events are forwarded to it from whichever
// connection is currently active. Events are sent non-blockingly and may be
// dropped if the consumer is slow.
func (a *ReconnectingAdapter) Subscribe(ctx context.Context, eventNames []string) (<-chan Message, error) {
	ch := make(chan Message, 16)
	a.mu.Lock()
	id := a.nextID
	a.nextID++
	a.subs[id] = &sub{events: append([]string(nil), eventNames...), ch: ch}
	a.events = unionEvents(a.subs)
	client := a.client
	parent := a.runCtx
	// If there's already a reader running, we need to restart it with the new event list
	restartReader := client != nil && parent != nil && a.readerCancel != nil
	a.mu.Unlock()

	if restartReader {
		a.mu.Lock()
		if a.readerCancel != nil {
			a.readerCancel()
			a.readerCancel = nil
		}
		a.mu.Unlock()
		go a.subscribeAndRead(parent, client)
	} else if client != nil && parent != nil {
		go a.subscribeAndRead(parent, client)
	}
	return ch, nil
}

// Close shuts down the adapter.
func (a *ReconnectingAdapter) Close() error {
	_ = a.Stop(context.Background())
	return nil
}

func (a *ReconnectingAdapter) run(ctx context.Context) {
	defer close(a.done)
	backoff := reconnectInitialBackoff
	for {
		if err := ctx.Err(); err != nil {
			a.setConnected(false)
			return
		}
		client, err := a.dial(ctx)
		if err != nil {
			a.logger.Warn("sway: dial failed, retrying", slog.Any("error", err))
			a.setConnected(false)
			a.waitBackoff(ctx, &backoff)
			continue
		}
		a.mu.Lock()
		old := a.client
		a.client = client
		if a.readerCancel != nil {
			a.readerCancel()
			a.readerCancel = nil
		}
		a.readerDone = nil
		a.mu.Unlock()
		if old != nil {
			_ = old.Close()
		}
		a.setConnected(true)
		a.publishConnected(client)
		a.logger.Info("sway: connected")
		if len(a.currentEvents()) > 0 {
			go a.subscribeAndRead(ctx, client)
		}
		select {
		case <-client.readerDone:
		case <-ctx.Done():
		}
		a.mu.Lock()
		if a.readerCancel != nil {
			a.readerCancel()
			a.readerCancel = nil
		}
		if a.client == client {
			a.client = nil
		}
		a.readerDone = nil
		a.mu.Unlock()
		a.setConnected(false)
		a.publishDisconnected("connection lost")
		a.logger.Warn("sway: disconnected, reconnecting", slog.Duration("backoff", backoff))
		backoff = reconnectInitialBackoff
		a.waitBackoff(ctx, &backoff)
	}
}

func (a *ReconnectingAdapter) currentEvents() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.events...)
}

func (a *ReconnectingAdapter) waitBackoff(ctx context.Context, backoff *time.Duration) {
	if *backoff <= 0 {
		*backoff = reconnectInitialBackoff
	}
	wait := *backoff
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
	*backoff = time.Duration(float64(*backoff) * reconnectMultiplier)
	if *backoff > reconnectMaxBackoff {
		*backoff = reconnectMaxBackoff
	}
}

func (a *ReconnectingAdapter) setConnected(v bool) {
	a.connected.Store(v)
}

func (a *ReconnectingAdapter) publishConnected(client *Client) {
	if a.bus == nil {
		return
	}
	// Get sway version for the event
	versionStr := ""
	if v, err := client.GetVersion(context.Background()); err == nil {
		versionStr = v.HumanReadable
	}
	socketPath := ""
	if p, err := SocketPath(context.Background()); err == nil {
		socketPath = p
	}
	ev := events.NewEvent(events.TypeSwayConnected, "sway", events.SwayConnected{
		Version:    events.Version,
		SocketPath: socketPath,
		VersionStr: versionStr,
	})
	_ = eventbus.Publish(a.bus, string(events.TypeSwayConnected), ev)
}

func (a *ReconnectingAdapter) publishDisconnected(reason string) {
	if a.bus == nil {
		return
	}
	ev := events.NewEvent(events.TypeSwayDisconnected, "sway", events.SwayDisconnected{
		Version: events.Version,
		Reason:  reason,
	})
	_ = eventbus.Publish(a.bus, string(events.TypeSwayDisconnected), ev)
}

func (a *ReconnectingAdapter) subscribeAndRead(parent context.Context, client *Client) {
	a.mu.Lock()
	eventNames := append([]string(nil), a.events...)
	if a.client != client || a.readerCancel != nil || len(eventNames) == 0 {
		a.mu.Unlock()
		return
	}
	readerCtx, cancel := context.WithCancel(parent)
	rd := make(chan struct{})
	a.readerCancel = cancel
	a.readerDone = rd
	a.mu.Unlock()

	cech, err := client.Subscribe(readerCtx, eventNames)
	if err != nil {
		a.logger.Warn("sway: subscribe failed", slog.Any("error", err))
		cancel()
		a.mu.Lock()
		a.readerCancel = nil
		a.readerDone = nil
		a.mu.Unlock()
		return
	}
	go a.eventReader(readerCtx, cech, rd)
}

func (a *ReconnectingAdapter) eventReader(ctx context.Context, cech <-chan Message, done chan struct{}) {
	defer close(done)
	for {
		select {
		case <-ctx.Done():
			return
		case m, ok := <-cech:
			if !ok {
				return
			}
			if !IsEvent(m.Type) {
				continue
			}
			// Handle shutdown event specially - trigger snapshot rebuild
			if m.Type == EventShutdown {
				a.logger.Info("sway: shutdown event received")
				a.publishDisconnected("sway shutdown")
				// The connection will be closed by sway, readerDone will be signaled
				return
			}
			a.mu.Lock()
			subs := make([]*sub, 0, len(a.subs))
			for _, s := range a.subs {
				subs = append(subs, s)
			}
			a.mu.Unlock()
			for _, s := range subs {
				if !a.eventWantedFor(m.Type, s.events) {
					continue
				}
				select {
				case s.ch <- m:
				default:
				}
			}
		}
	}
}

func unionEvents(subs map[uint64]*sub) []string {
	seen := make(map[string]bool)
	var out []string
	for _, s := range subs {
		for _, e := range s.events {
			if !seen[e] {
				seen[e] = true
				out = append(out, e)
			}
		}
	}
	return out
}

func (a *ReconnectingAdapter) eventWantedFor(t MessageType, eventNames []string) bool {
	name, ok := eventName[t]
	if !ok {
		return false
	}
	for _, e := range eventNames {
		if e == name {
			return true
		}
	}
	return false
}

// Compile-time interface checks.
var (
	_ registry.Service = (*ReconnectingAdapter)(nil)
	_ Adapter          = (*ReconnectingAdapter)(nil)
)
