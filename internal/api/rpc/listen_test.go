package rpc

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func testSocketPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "i3-next", "i3-next.sock")
}

func closeLater(t *testing.T, ln net.Listener) {
	t.Helper()
	t.Cleanup(func() { _ = ln.Close() })
}

func dialOK(t *testing.T, path string) {
	t.Helper()
	d := net.Dialer{}
	c, err := d.DialContext(context.Background(), "unix", path)
	if err != nil {
		t.Fatalf("dial %s: %v", path, err)
	}
	_ = c.Close()
}

func TestListenPermissions(t *testing.T) {
	// A permissive umask must not leak into the socket or its directory.
	old := syscall.Umask(0)
	t.Cleanup(func() { syscall.Umask(old) })

	path := testSocketPath(t)
	ln, err := Listen(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	closeLater(t, ln)

	di, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if got := di.Mode().Perm(); got != 0o700 {
		t.Fatalf("dir mode = %o, want 700", got)
	}
	si, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if si.Mode()&os.ModeSocket == 0 {
		t.Fatalf("%s is not a socket", path)
	}
	if got := si.Mode().Perm(); got != 0o600 {
		t.Fatalf("socket mode = %o, want 600", got)
	}
	dialOK(t, path)
}

func TestListenTightensExistingDir(t *testing.T) {
	path := testSocketPath(t)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // deliberately wide to test tightening
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0o755); err != nil { //nolint:gosec // deliberately wide to test tightening
		t.Fatal(err)
	}
	ln, err := Listen(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	closeLater(t, ln)
	di, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := di.Mode().Perm(); got != 0o700 {
		t.Fatalf("dir mode = %o, want 700", got)
	}
}

func TestListenRemovesStaleSocket(t *testing.T) {
	path := testSocketPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	// Leave a socket file behind with nobody listening (a crashed daemon).
	var lc net.ListenConfig
	raw, err := lc.Listen(context.Background(), "unix", path)
	if err != nil {
		t.Fatal(err)
	}
	ul, ok := raw.(*net.UnixListener)
	if !ok {
		t.Fatalf("listener is %T, want *net.UnixListener", raw)
	}
	ul.SetUnlinkOnClose(false)
	_ = ul.Close()
	if _, err := os.Lstat(path); err != nil {
		t.Fatalf("stale socket missing: %v", err)
	}

	ln, err := Listen(context.Background(), path)
	if err != nil {
		t.Fatalf("Listen over stale socket: %v", err)
	}
	closeLater(t, ln)
	dialOK(t, path)
}

func TestListenKeepsLiveSocket(t *testing.T) {
	path := testSocketPath(t)
	first, err := Listen(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	closeLater(t, first)

	second, err := Listen(context.Background(), path)
	if err == nil {
		_ = second.Close()
		t.Fatal("second Listen succeeded while the first is alive")
	}
	if !errors.Is(err, ErrAddressInUse) {
		t.Fatalf("err = %v, want ErrAddressInUse", err)
	}
	dialOK(t, path) // the first listener is untouched
}

func TestListenRefusesNonSocket(t *testing.T) {
	path := testSocketPath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("precious"), 0o600); err != nil {
		t.Fatal(err)
	}
	if ln, err := Listen(context.Background(), path); err == nil {
		_ = ln.Close()
		t.Fatal("Listen replaced a regular file")
	}
	got, err := os.ReadFile(path) //nolint:gosec // test path under t.TempDir
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "precious" {
		t.Fatalf("file content changed: %q", got)
	}
}

func TestListenRejectsBadPaths(t *testing.T) {
	if _, err := Listen(context.Background(), "relative/i3-next.sock"); err == nil {
		t.Fatal("relative path accepted")
	}
	long := filepath.Join(t.TempDir(), strings.Repeat("a", 120), "i3-next.sock")
	if _, err := Listen(context.Background(), long); err == nil {
		t.Fatal("over-long path accepted")
	}
}

func TestListenRestoresUmask(t *testing.T) {
	old := syscall.Umask(0o022)
	t.Cleanup(func() { syscall.Umask(old) })

	ln, err := Listen(context.Background(), testSocketPath(t))
	if err != nil {
		t.Fatal(err)
	}
	closeLater(t, ln)
	if got := syscall.Umask(0o022); got != 0o022 {
		t.Fatalf("umask after Listen = %o, want 022", got)
	}
}

func TestListenCloseRemovesSocket(t *testing.T) {
	path := testSocketPath(t)
	ln, err := Listen(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("socket still present after Close: %v", err)
	}
}
