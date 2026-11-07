package workspace

import (
	"context"
	"errors"
	"fmt"

	"i3_next/internal/adapters/sway"
	"i3_next/internal/api/contracts"
	"i3_next/internal/api/events"
	"i3_next/internal/core/eventbus"
	"i3_next/internal/core/state"
)

// Snapshot is the immutable workspace/window-manager snapshot.
type Snapshot struct {
	Workspaces []sway.Workspace `json:"workspaces"`
	Outputs    []sway.Output    `json:"outputs"`
	Tree       *sway.Node       `json:"tree,omitempty"`
}

// Service implements the workspace/window-manager service.
type Service struct {
	name    string
	adapter sway.Adapter
	bus     *eventbus.Bus
	state   *state.State

	cancel context.CancelFunc
	done   chan struct{}
}

// New creates a new workspace service.
func New(adapter sway.Adapter, bus *eventbus.Bus) *Service {
	return &Service{
		name:    "workspace",
		adapter: adapter,
		bus:     bus,
		state:   state.NewState(Snapshot{}),
		done:    make(chan struct{}),
	}
}

func (s *Service) Name() string { return s.name }

// Snapshot returns the current snapshot.
func (s *Service) Snapshot() state.Snapshot { return s.state.Snapshot() }

// State returns the underlying state container.
func (s *Service) State() *state.State { return s.state }

func (s *Service) Start(ctx context.Context) error {
	if s.adapter == nil {
		return nil
	}
	if err := s.rebuild(ctx); err != nil {
		return err
	}
	subCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	ch, err := s.adapter.Subscribe(subCtx, []string{"workspace", "window", "output", "shutdown"})
	if err != nil {
		cancel()
		return err
	}
	go s.loop(subCtx, ch)
	return nil
}

func (s *Service) Stop(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}
	select {
	case <-s.done:
	case <-ctx.Done():
	}
	return nil
}

func (s *Service) Health(ctx context.Context) contracts.ServiceInfo {
	info := contracts.ServiceInfo{Name: s.name, State: "running", Health: "ok"}
	return info
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
	s.state.Set(snap)
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
		_ = s.bus.Publish(string(events.TypeWorkspaceChanged), ev, eventbus.State())
	}
}

func (s *Service) publishOutputEvents(out []sway.Output) {
	if s.bus == nil {
		return
	}
	for _, o := range out {
		ev := events.NewEvent(events.TypeOutputChanged, s.name, events.OutputChanged{
			Name:      o.Name,
			Width:     o.Rect.Width,
			Height:    o.Rect.Height,
			Scale:     o.Scale,
			Connected: true,
		})
		_ = s.bus.Publish(string(events.TypeOutputChanged), ev, eventbus.State())
	}
}

func (s *Service) loop(ctx context.Context, ch <-chan sway.Message) {
	defer close(s.done)
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				if s.bus != nil {
					ev := events.NewEvent(events.TypeSwayDisconnected, s.name, events.SwayDisconnected{Reason: "channel closed"})
					_ = s.bus.Publish(string(events.TypeSwayDisconnected), ev)
				}
				return
			}
			_ = msg
			_ = s.rebuild(ctx)
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

// FocusWorkspace focuses the workspace with the given name.
func (s *Service) FocusWorkspace(ctx context.Context, name string) error {
	if err := sway.ValidateWorkspaceName(name); err != nil {
		return err
	}
	return s.runCommand(ctx, fmt.Sprintf("workspace %s", name))
}

// MoveWindowToWorkspace moves the window matching app_id to the workspace.
func (s *Service) MoveWindowToWorkspace(ctx context.Context, appID, workspace string) error {
	if err := sway.ValidateAppID(appID); err != nil {
		return err
	}
	if err := sway.ValidateWorkspaceName(workspace); err != nil {
		return err
	}
	return s.runCommand(ctx, fmt.Sprintf("[app_id=\"%s\"] move window to workspace %s", appID, workspace))
}

// CloseWindow closes the window matching app_id, or the focused window when appID is empty.
func (s *Service) CloseWindow(ctx context.Context, appID string) error {
	if appID != "" {
		if err := sway.ValidateAppID(appID); err != nil {
			return err
		}
		return s.runCommand(ctx, fmt.Sprintf("[app_id=\"%s\"] kill", appID))
	}
	return s.runCommand(ctx, "kill")
}

// FloatingEnable enables floating mode for the window matching app_id.
func (s *Service) FloatingEnable(ctx context.Context, appID string) error {
	if err := sway.ValidateAppID(appID); err != nil {
		return err
	}
	return s.runCommand(ctx, fmt.Sprintf("[app_id=\"%s\"] floating enable", appID))
}

// FloatingDisable disables floating mode for the window matching app_id.
func (s *Service) FloatingDisable(ctx context.Context, appID string) error {
	if err := sway.ValidateAppID(appID); err != nil {
		return err
	}
	return s.runCommand(ctx, fmt.Sprintf("[app_id=\"%s\"] floating disable", appID))
}

// Layout sets the layout of the focused container. layout must be one of
// splitv, splith, stacking, tabbed, or "toggle split".
func (s *Service) Layout(ctx context.Context, layout string) error {
	switch layout {
	case "splitv", "splith", "stacking", "tabbed", "toggle split":
	default:
		return fmt.Errorf("%w: unsupported layout %q", sway.ErrInvalidSwayInput, layout)
	}
	return s.runCommand(ctx, fmt.Sprintf("layout %s", layout))
}

// MoveToOutput moves the window matching app_id to the output.
func (s *Service) MoveToOutput(ctx context.Context, appID, output string) error {
	if err := sway.ValidateAppID(appID); err != nil {
		return err
	}
	if err := sway.ValidateOutputName(output); err != nil {
		return err
	}
	return s.runCommand(ctx, fmt.Sprintf("[app_id=\"%s\"] move window to output %s", appID, output))
}
