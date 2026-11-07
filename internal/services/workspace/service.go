package workspace

import (
	"context"

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
