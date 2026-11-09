package workspace

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"i3_next/internal/adapters/sway"
	"i3_next/internal/api/contracts"
	"i3_next/internal/api/events"
	"i3_next/internal/core/eventbus"
	"i3_next/internal/core/state"
)

// connectable is implemented by adapters that can report their connection
// state (for example the reconnecting adapter).
type connectable interface {
	Connected() bool
}

// Snapshot is the immutable workspace/window-manager snapshot.
type Snapshot struct {
	Workspaces []sway.Workspace `json:"workspaces"`
	Outputs    []sway.Output    `json:"outputs"`
	Tree       *sway.Node       `json:"tree,omitempty"`
}

// Service implements the workspace/window-manager service.
type Service struct {
	logger  *slog.Logger
	name    string
	adapter sway.Adapter
	bus     *eventbus.Bus
	state   *state.State

	cancel        context.CancelFunc
	done          chan struct{}
	swayCancel    context.CancelFunc
	swayDone      chan struct{}
	rebuildCancel context.CancelFunc
}

func New(adapter sway.Adapter, bus *eventbus.Bus) *Service {
	return &Service{
		logger:  slog.New(slog.NewTextHandler(nil, nil)),
		name:    "workspace",
		adapter: adapter,
		bus:     bus,
		state:   state.NewState(Snapshot{}),
		done:    make(chan struct{}),
	}
}

func (s *Service) Name() string { return s.name }

func (s *Service) DependsOn() []string { return []string{"sway"} }

func (s *Service) Optional() bool { return true }

func (s *Service) Snapshot() state.Snapshot { return s.state.Snapshot() }

func (s *Service) State() *state.State { return s.state }

func (s *Service) Start(ctx context.Context) error {
	if s.adapter == nil {
		return nil
	}
	if c, ok := s.adapter.(connectable); ok {
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if c.Connected() {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	if err := s.rebuild(ctx); err != nil {
		s.publishState(contracts.ServiceFailed, err.Error())
		return err
	}
	s.publishState(contracts.ServiceRunning, "")

	subCtx, cancel := context.WithCancel(ctx)
	s.swayCancel = cancel
	swayCh, err := s.adapter.Subscribe(subCtx, []string{"workspace", "window", "output", "shutdown"})
	if err != nil {
		cancel()
		s.publishState(contracts.ServiceFailed, err.Error())
		return err
	}
	s.swayDone = make(chan struct{})
	go s.swayLoop(subCtx, swayCh)

	rebuildCtx, rebuildCancel := context.WithCancel(ctx)
	s.rebuildCancel = rebuildCancel
	go s.rebuildLoop(rebuildCtx)

	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if s.swayCancel != nil {
		s.swayCancel()
	}
	if s.rebuildCancel != nil {
		s.rebuildCancel()
	}
	if s.cancel != nil {
		s.cancel()
	}
	select {
	case <-s.done:
	case <-ctx.Done():
	}
	s.publishState(contracts.ServiceStopped, "")
	return nil
}

func (s *Service) Health(ctx context.Context) contracts.ServiceInfo {
	info := contracts.ServiceInfo{Name: s.name, State: "running", Health: "ok"}
	if c, ok := s.adapter.(connectable); ok && !c.Connected() {
		info.State = contracts.ServiceDegraded
		info.Health = contracts.HealthWarn
	}
	return info
}

func (s *Service) publishState(st contracts.ServiceState, reason string) {
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
}

func (s *Service) rebuild(ctx context.Context) error {
	ws, err := s.adapter.GetWorkspaces(ctx)
	if err != nil {
		return err
	}
	out, err := s.adapter.GetOutputs(ctx)
	if err != nil {
		return err
	}
	tree, _ := s.adapter.GetTree(ctx)
	snap := Snapshot{Workspaces: ws, Outputs: out, Tree: tree}
	s.state.Update(snap)
	s.publishWorkspaceEvents(ws)
	s.publishOutputEvents(out)
	return nil
}

func (s *Service) publishWorkspaceEvents(ws []sway.Workspace) {
	if s.bus == nil {
		return
	}
	for _, w := range ws {
		if !w.Focused {
			continue
		}
		ev := events.NewEvent(events.TypeWorkspaceChanged, s.name, events.WorkspaceChanged{
			Number:  w.Num,
			Name:    w.Name,
			Output:  w.Output,
			Focused: w.Focused,
		})
		_ = eventbus.Publish(s.bus, string(events.TypeWorkspaceChanged), ev, eventbus.State())
	}
}

func (s *Service) publishOutputEvents(out []sway.Output) {
	if s.bus == nil {
		return
	}
	for i := range out {
		o := out[i]
		ev := events.NewEvent(events.TypeOutputChanged, s.name, events.OutputChanged{
			Name:      o.Name,
			Width:     o.Rect.Width,
			Height:    o.Rect.Height,
			Scale:     o.Scale,
			Connected: true,
		})
		_ = eventbus.Publish(s.bus, string(events.TypeOutputChanged), ev, eventbus.State())
	}
}

func (s *Service) swayLoop(ctx context.Context, ch <-chan sway.Message) {
	defer close(s.swayDone)
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				s.publishState(contracts.ServiceDegraded, "sway disconnected")
				return
			}
			_ = msg
			if err := s.rebuild(ctx); err != nil {
				s.publishState(contracts.ServiceDegraded, err.Error())
			}
		}
	}
}

func (s *Service) rebuildLoop(ctx context.Context) {
	connectedCh := make(chan events.Event, 1)
	disconnectedCh := make(chan events.Event, 1)
	cancelConn := eventbus.SubscribeFunc[events.Event](s.bus, events.TopicSwayConnected, func(ev events.Event) {
		select {
		case connectedCh <- ev:
		default:
		}
	})
	cancelDisc := eventbus.SubscribeFunc[events.Event](s.bus, events.TopicSwayDisconnected, func(ev events.Event) {
		select {
		case disconnectedCh <- ev:
		default:
		}
	})
	defer cancelConn()
	defer cancelDisc()

	for {
		select {
		case <-ctx.Done():
			return
		case <-connectedCh:
			s.logger.Info("sway reconnected, rebuilding snapshot")
			if err := s.rebuild(ctx); err != nil {
				s.logger.Error("failed to rebuild snapshot on reconnect", slog.Any("error", err))
				s.publishState(contracts.ServiceDegraded, err.Error())
			} else {
				s.publishState(contracts.ServiceRunning, "")
			}
		case ev := <-disconnectedCh:
			if payload, ok := ev.Payload.(events.SwayDisconnected); ok && payload.Reason == "sway shutdown" {
				s.logger.Info("sway shutdown, marking as degraded")
				s.publishState(contracts.ServiceDegraded, "sway shutdown")
			}
		}
	}
}

func (s *Service) runCommand(ctx context.Context, command string) error {
	if s.adapter == nil {
		return errors.New("workspace: sway adapter not available")
	}
	_, err := s.adapter.RunCommand(ctx, command)
	return err
}

func (s *Service) FocusWorkspace(ctx context.Context, name string) error {
	if err := sway.ValidateWorkspaceName(name); err != nil {
		return err
	}
	return s.runCommand(ctx, fmt.Sprintf("workspace %s", name))
}

func (s *Service) MoveWindowToWorkspace(ctx context.Context, appID, workspace string) error {
	if err := sway.ValidateAppID(appID); err != nil {
		return err
	}
	if err := sway.ValidateWorkspaceName(workspace); err != nil {
		return err
	}
	return s.runCommand(ctx, fmt.Sprintf("[app_id=%q] move window to workspace %s", appID, workspace))
}

func (s *Service) CloseWindow(ctx context.Context, appID string) error {
	if appID != "" {
		if err := sway.ValidateAppID(appID); err != nil {
			return err
		}
		return s.runCommand(ctx, fmt.Sprintf("[app_id=%q] kill", appID))
	}
	return s.runCommand(ctx, "kill")
}

func (s *Service) FloatingEnable(ctx context.Context, appID string) error {
	if err := sway.ValidateAppID(appID); err != nil {
		return err
	}
	return s.runCommand(ctx, fmt.Sprintf("[app_id=%q] floating enable", appID))
}

func (s *Service) FloatingDisable(ctx context.Context, appID string) error {
	if err := sway.ValidateAppID(appID); err != nil {
		return err
	}
	return s.runCommand(ctx, fmt.Sprintf("[app_id=%q] floating disable", appID))
}

func (s *Service) Layout(ctx context.Context, layout string) error {
	switch layout {
	case "splitv", "splith", "stacking", "tabbed", "toggle split":
	default:
		return fmt.Errorf("%w: unsupported layout %q", sway.ErrInvalidSwayInput, layout)
	}
	return s.runCommand(ctx, fmt.Sprintf("layout %s", layout))
}

func (s *Service) MoveToOutput(ctx context.Context, appID, output string) error {
	if err := sway.ValidateAppID(appID); err != nil {
		return err
	}
	if err := sway.ValidateOutputName(output); err != nil {
		return err
	}
	return s.runCommand(ctx, fmt.Sprintf("[app_id=%q] move window to output %s", appID, output))
}
