package rpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const (
	// dirMode is the mode of the directory that holds the control socket.
	dirMode os.FileMode = 0o700
	// socketMode is the mode of the control socket itself.
	socketMode os.FileMode = 0o600
	// socketUmask clears every group/other bit, so the socket is created 0600.
	socketUmask = 0o177
	// maxSocketPath is the longest usable AF_UNIX path (sun_path is 108 bytes
	// including the trailing NUL).
	maxSocketPath = 107
	// probeTimeout bounds the liveness probe of an existing socket.
	probeTimeout = 500 * time.Millisecond
)

// ErrAddressInUse is returned when another process is already serving the
// control socket.
var ErrAddressInUse = errors.New("rpc: control socket already in use")

// umaskMu serialises the process-wide umask change done around Listen.
var umaskMu sync.Mutex

// Listen creates the control socket at path (an absolute path, normally
// config.SocketPath()) and returns its listener.
//
//   - The parent directory is created (or tightened) to mode 0700 and must be
//     a real directory owned by the current user.
//   - The umask is set to 0177 around the bind so the socket is never visible
//     with wider permissions, and the mode is then enforced as 0600.
//   - An existing socket is removed only when connecting to it is refused
//     (nobody is listening). A live socket yields ErrAddressInUse, and a path
//     that is not a socket is never touched.
//
// Closing the returned listener removes the socket file.
func Listen(ctx context.Context, path string) (net.Listener, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("rpc: socket path %q must be absolute", path)
	}
	if len(path) > maxSocketPath {
		return nil, fmt.Errorf("rpc: socket path too long (%d > %d bytes): %s", len(path), maxSocketPath, path)
	}
	if err := prepareDir(filepath.Dir(path)); err != nil {
		return nil, err
	}
	if err := removeStale(ctx, path); err != nil {
		return nil, err
	}
	ln, err := listenRestricted(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("rpc: listen %s: %w", path, err)
	}
	if err := os.Chmod(path, socketMode); err != nil {
		_ = ln.Close()
		return nil, fmt.Errorf("rpc: chmod %s: %w", path, err)
	}
	return ln, nil
}

// prepareDir makes sure dir exists, is a directory owned by the current user
// and has mode 0700.
func prepareDir(dir string) error {
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return fmt.Errorf("rpc: create socket dir: %w", err)
	}
	fi, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("rpc: stat socket dir: %w", err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("rpc: socket dir %s is not a directory", dir)
	}
	if st, ok := fi.Sys().(*syscall.Stat_t); ok && st.Uid != uint32(os.Geteuid()) { //nolint:gosec // effective UID is never negative
		return fmt.Errorf("rpc: socket dir %s is owned by uid %d, not the current user", dir, st.Uid)
	}
	if err := os.Chmod(dir, dirMode); err != nil {
		return fmt.Errorf("rpc: chmod socket dir: %w", err)
	}
	return nil
}

// removeStale deletes path only if it is a socket nobody is listening on.
func removeStale(ctx context.Context, path string) error {
	fi, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("rpc: stat %s: %w", path, err)
	}
	if fi.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("rpc: %s exists and is not a socket; refusing to remove it", path)
	}
	d := net.Dialer{Timeout: probeTimeout}
	conn, err := d.DialContext(ctx, "unix", path)
	if err == nil {
		_ = conn.Close()
		return fmt.Errorf("%w: %s", ErrAddressInUse, path)
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		// Timeout, permission denied, ...: we cannot prove the socket is stale.
		return fmt.Errorf("rpc: probing %s: %w", path, err)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("rpc: remove stale socket %s: %w", path, err)
	}
	return nil
}

// listenRestricted binds path with a restrictive umask.
func listenRestricted(ctx context.Context, path string) (net.Listener, error) {
	umaskMu.Lock()
	defer umaskMu.Unlock()
	old := syscall.Umask(socketUmask)
	defer syscall.Umask(old)
	var lc net.ListenConfig
	return lc.Listen(ctx, "unix", path)
}
