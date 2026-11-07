package sway

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreerrors "i3_next/internal/core/errors"
)

func TestSocketPathFromEnv(t *testing.T) {
	t.Setenv("SWAYSOCK", "/tmp/test-sway.sock")
	p, err := SocketPath(context.Background())
	if err != nil {
		t.Fatalf("SocketPath: %v", err)
	}
	if p != "/tmp/test-sway.sock" {
		t.Fatalf("got %q, want /tmp/test-sway.sock", p)
	}
}

func TestSocketPathTrimsEnv(t *testing.T) {
	t.Setenv("SWAYSOCK", "  /tmp/test-sway.sock\n")
	p, err := SocketPath(context.Background())
	if err != nil {
		t.Fatalf("SocketPath: %v", err)
	}
	if p != "/tmp/test-sway.sock" {
		t.Fatalf("got %q", p)
	}
}

func TestSocketPathFallbackSwaymsg(t *testing.T) {
	t.Setenv("SWAYSOCK", "")
	dir := t.TempDir()
	sock := filepath.Join(dir, "ipc.sock")
	bin := filepath.Join(dir, "swaymsg")
	script := "#!/bin/sh\necho '" + sock + "'\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	p, err := socketPathFromSwaymsg(context.Background(), bin)
	if err != nil {
		t.Fatalf("socketPathFromSwaymsg: %v", err)
	}
	if p != sock {
		t.Fatalf("got %q, want %q", p, sock)
	}
}

func TestSocketPathSwaymsgEmpty(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "swaymsg")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := socketPathFromSwaymsg(context.Background(), bin)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrSocketNotFound) {
		t.Fatalf("err %v not ErrSocketNotFound", err)
	}
	if !errors.Is(err, coreerrors.ErrUnavailable) {
		t.Fatalf("err %v not ErrUnavailable", err)
	}
	if !strings.Contains(err.Error(), "sway socket not found") {
		t.Fatalf("err %q missing clear message", err)
	}
}

func TestSocketPathSwaymsgFailure(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "swaymsg")
	script := "#!/bin/sh\necho 'Unable to retrieve socket path' >&2\nexit 2\n"
	if err := os.WriteFile(bin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := socketPathFromSwaymsg(context.Background(), bin)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrSocketNotFound) {
		t.Fatalf("err %v not ErrSocketNotFound", err)
	}
	if !strings.Contains(err.Error(), "Unable to retrieve socket path") {
		t.Fatalf("err %q missing stderr detail", err)
	}
}

func TestSocketPathMissingBoth(t *testing.T) {
	t.Setenv("SWAYSOCK", "")
	// Point PATH to an empty dir so swaymsg is not found.
	t.Setenv("PATH", t.TempDir())
	_, err := SocketPath(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrSocketNotFound) {
		t.Fatalf("err %v not ErrSocketNotFound", err)
	}
	if !strings.Contains(err.Error(), "sway socket not found") {
		t.Fatalf("err %q missing clear message", err)
	}
}
