package workspace

import (
	"context"
	"errors"
	"testing"

	"i3_next/internal/adapters/sway"
)

func TestServiceOperationsBuildSafeCommands(t *testing.T) {
	var got []string
	fake := &sway.FakeAdapter{
		RunCommandFn: func(ctx context.Context, command string) (sway.Message, error) {
			got = append(got, command)
			return sway.Message{}, nil
		},
	}
	s := New(fake, nil)

	if err := s.FocusWorkspace(context.Background(), "web"); err != nil {
		t.Fatal(err)
	}
	if err := s.MoveWindowToWorkspace(context.Background(), "foot", "web"); err != nil {
		t.Fatal(err)
	}
	if err := s.CloseWindow(context.Background(), "foot"); err != nil {
		t.Fatal(err)
	}
	if err := s.CloseWindow(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if err := s.FloatingEnable(context.Background(), "foot"); err != nil {
		t.Fatal(err)
	}
	if err := s.FloatingDisable(context.Background(), "foot"); err != nil {
		t.Fatal(err)
	}
	if err := s.Layout(context.Background(), "splitv"); err != nil {
		t.Fatal(err)
	}
	if err := s.Layout(context.Background(), "toggle split"); err != nil {
		t.Fatal(err)
	}
	if err := s.MoveToOutput(context.Background(), "foot", "eDP-1"); err != nil {
		t.Fatal(err)
	}

	want := []string{
		"workspace web",
		`[app_id="foot"] move window to workspace web`,
		`[app_id="foot"] kill`,
		"kill",
		`[app_id="foot"] floating enable`,
		`[app_id="foot"] floating disable`,
		"layout splitv",
		"layout toggle split",
		`[app_id="foot"] move window to output eDP-1`,
	}
	if len(got) != len(want) {
		t.Fatalf("got %d commands, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("command %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestServiceOperationsRejectInjection(t *testing.T) {
	var called bool
	fake := &sway.FakeAdapter{
		RunCommandFn: func(ctx context.Context, command string) (sway.Message, error) {
			called = true
			return sway.Message{}, nil
		},
	}
	s := New(fake, nil)

	// Workspace name injection must be rejected and never reach the adapter.
	for _, name := range []string{"; exec foot", ";exec foot", "web; exec foot", "web\nexec foot", `web"; exec foot`, ""} {
		called = false
		if err := s.FocusWorkspace(context.Background(), name); err == nil {
			t.Errorf("FocusWorkspace(%q) did not reject injection", name)
		}
		if called {
			t.Errorf("FocusWorkspace(%q) reached the adapter despite invalid input", name)
		}
	}

	// app_id injection must be rejected and never reach the adapter.
	for _, appID := range []string{`"; exec foot`, `foot"; exec foot`, "; exec foot", ""} {
		called = false
		if err := s.MoveWindowToWorkspace(context.Background(), appID, "web"); err == nil {
			t.Errorf("MoveWindowToWorkspace(app_id=%q) did not reject injection", appID)
		}
		if called {
			t.Errorf("MoveWindowToWorkspace(app_id=%q) reached the adapter despite invalid input", appID)
		}
	}

	// Unknown layout must be rejected.
	if err := s.Layout(context.Background(), "bad; exec foot"); err == nil {
		t.Errorf("Layout(%q) did not reject invalid layout", "bad; exec foot")
	}
	if called {
		t.Errorf("Layout reached the adapter despite invalid layout")
	}

	// Adapter errors must propagate.
	fake.RunCommandFn = func(ctx context.Context, command string) (sway.Message, error) {
		return sway.Message{}, errors.New("sway: command failed")
	}
	if err := s.FocusWorkspace(context.Background(), "web"); err == nil {
		t.Errorf("FocusWorkspace did not propagate adapter error")
	}
}
