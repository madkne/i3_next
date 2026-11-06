package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"sync"
	"syscall"
	"time"

	"i3_next/internal/core/config"
	"i3_next/internal/version"
)

// Errors returned by the bootstrap package.
var (
	ErrAlreadyRunning = errors.New("already running")
	ErrNotRunning     = errors.New("not running")
	ErrBootstrap      = errors.New("bootstrap failed")
)

// Option configures an App.
type Option func(*App)

// WithRuntimeDir overrides the runtime directory used for locks and state.
func WithRuntimeDir(dir string) Option {
	return func(a *App) {
		a.runtimeDir = dir
	}
}

// WithShutdownTimeout sets the graceful shutdown timeout.
func WithShutdownTimeout(d time.Duration) Option {
	return func(a *App) {
		if d > 0 {
			a.shutdownTimeout = d
		}
	}
}

// WithService registers a service during construction.
func WithService(s any) Option {
	return func(a *App) {
		a.services = append(a.services, s)
	}
}

// WithStartupHook registers a startup hook during construction.
func WithStartupHook(fn func(context.Context) error) Option {
	return func(a *App) {
		if fn != nil {
			a.startupHooks = append(a.startupHooks, fn)
		}
	}
}

// App is the desktop core application container.
type App struct {
	logger          *slog.Logger
	runtimeDir      string
	shutdownTimeout time.Duration

	mu           sync.Mutex
	running      bool
	services     []any
	startupHooks []func(context.Context) error

	lockFile *os.File
}

// New creates a new App.
func New(logger *slog.Logger, opts ...Option) (*App, error) {
	if logger == nil {
		return nil, ErrLoggerRequired
	}

	a := &App{
		logger:          logger,
		runtimeDir:      config.RuntimeDir(),
		shutdownTimeout: 10 * time.Second,
	}

	for _, opt := range opts {
		if opt != nil {
			opt(a)
		}
	}

	return a, nil
}

// Register adds a component to the bootstrap lifecycle.
func (a *App) Register(service any) error {
	if service == nil {
		return fmt.Errorf("%w: nil service", ErrBootstrap)
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.running {
		return ErrAlreadyRunning
	}
	a.services = append(a.services, service)
	return nil
}

// AddService is an alias for Register.
func (a *App) AddService(service any) error { return a.Register(service) }

// Add is an alias for Register.
func (a *App) Add(service any) error { return a.Register(service) }

// Use is an alias for Register.
func (a *App) Use(service any) error { return a.Register(service) }

// AddStartupHook registers a startup hook.
func (a *App) AddStartupHook(fn func(context.Context) error) {
	if fn == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.startupHooks = append(a.startupHooks, fn)
}

// Run bootstraps, runs, and shuts down the application.
func (a *App) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return ErrAlreadyRunning
	}
	a.running = true
	services := append([]any(nil), a.services...)
	hooks := append([]func(context.Context) error(nil), a.startupHooks...)
	a.mu.Unlock()

	logger := a.logger

	cleanup := func() {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
	}

	// Bootstrap runtime directories.
	if err := a.bootstrapDirs(); err != nil {
		cleanup()
		return fmt.Errorf("%w: %v", ErrBootstrap, err)
	}

	// Acquire single-instance lock.
	lockFile, err := a.acquireLock()
	if err != nil {
		cleanup()
		return err
	}
	a.mu.Lock()
	a.lockFile = lockFile
	a.mu.Unlock()

	releaseLock := func() {
		a.mu.Lock()
		f := a.lockFile
		a.lockFile = nil
		a.mu.Unlock()
		if f != nil {
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			_ = f.Close()
		}
	}

	// Run startup hooks.
	for _, hook := range hooks {
		if err := hook(ctx); err != nil {
			releaseLock()
			cleanup()
			return fmt.Errorf("%w: startup hook: %v", ErrBootstrap, err)
		}
	}

	// Start services in registration order.
	started := make([]int, 0, len(services))
	for i, svc := range services {
		ok, err := callLifecycle(ctx, svc, "Start")
		if err != nil {
			// Roll back already started services.
			for j := len(started) - 1; j >= 0; j-- {
				idx := started[j]
				stopCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
				_, _ = callLifecycle(stopCtx, services[idx], "Stop")
				cancel()
			}
			releaseLock()
			cleanup()
			return fmt.Errorf("%w: start %s: %v", ErrBootstrap, componentName(svc), err)
		}
		if ok {
			started = append(started, i)
		}
	}

	logger.Info("starting i3_next desktop core",
		slog.String("version", version.Version),
		slog.Int("services", len(services)),
	)

	// Wait for shutdown signal or context cancellation.
	<-ctx.Done()

	logger.Info("shutting down i3_next")

	// Stop services in reverse order.
	stopCtx, cancel := context.WithTimeout(context.Background(), a.shutdownTimeout)
	defer cancel()
	for i := len(services) - 1; i >= 0; i-- {
		_, _ = callLifecycle(stopCtx, services[i], "Stop")
	}

	releaseLock()
	cleanup()
	return nil
}

func (a *App) bootstrapDirs() error {
	dirs := []string{
		a.runtimeDir,
		config.ConfigDir(),
		config.DataDir(),
		config.StateDir(),
		config.CacheDir(),
	}
	for _, d := range dirs {
		if d == "" {
			continue
		}
		mode := os.FileMode(0o755)
		if d == a.runtimeDir || d == config.ConfigDir() || d == config.DataDir() || d == config.StateDir() {
			mode = 0o700
		}
		if err := os.MkdirAll(d, mode); err != nil {
			return err
		}
		if err := os.Chmod(d, mode); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) acquireLock() (*os.File, error) {
	if err := os.MkdirAll(a.runtimeDir, 0o700); err != nil {
		return nil, err
	}
	if err := os.Chmod(a.runtimeDir, 0o700); err != nil {
		return nil, err
	}
	lockPath := filepath.Join(a.runtimeDir, "i3-next.lock")
	for attempts := 0; attempts < 8; attempts++ {
		f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			return nil, err
		}
		// Re-check the inode after open: if the path was replaced or
		// unlinked in the meantime, do not lock the stale inode.
		fi, err := f.Stat()
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		pi, err := os.Stat(lockPath)
		if err != nil {
			_ = f.Close()
			return nil, err
		}
		if !os.SameFile(fi, pi) {
			_ = f.Close()
			continue
		}
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			_ = f.Close()
			if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
				return nil, ErrAlreadyRunning
			}
			return nil, err
		}
		// Re-check again after locking to avoid locking a deleted inode.
		fi2, err := f.Stat()
		if err != nil {
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			_ = f.Close()
			return nil, err
		}
		pi2, err := os.Stat(lockPath)
		if err != nil || !os.SameFile(fi2, pi2) {
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			_ = f.Close()
			continue
		}
		_ = f.Truncate(0)
		_, _ = f.WriteString(strconv.Itoa(os.Getpid()))
		_ = f.Sync()
		return f, nil
	}
	return nil, ErrAlreadyRunning
}

// callLifecycle invokes a named lifecycle method if present.
func callLifecycle(ctx context.Context, v any, name string) (bool, error) {
	val := reflect.ValueOf(v)
	if !val.IsValid() {
		return false, nil
	}
	method := val.MethodByName(name)
	if !method.IsValid() {
		return false, nil
	}
	mt := method.Type()

	var args []reflect.Value
	if mt.NumIn() == 1 {
		param := mt.In(0)
		ctxType := reflect.TypeOf((*context.Context)(nil)).Elem()
		if !param.Implements(ctxType) {
			return false, fmt.Errorf("unexpected parameter type %s", param)
		}
		args = append(args, reflect.ValueOf(ctx))
	} else if mt.NumIn() != 0 {
		return false, fmt.Errorf("unexpected parameter count %d", mt.NumIn())
	}

	results := method.Call(args)
	if mt.NumOut() == 0 {
		return true, nil
	}
	last := results[mt.NumOut()-1]
	if err, ok := last.Interface().(error); ok {
		return true, err
	}
	return true, nil
}

func componentName(v any) string {
	val := reflect.ValueOf(v)
	if val.IsValid() {
		if m := val.MethodByName("Name"); m.IsValid() && m.Type().NumIn() == 0 && m.Type().NumOut() >= 1 {
			res := m.Call(nil)
			if s, ok := res[0].Interface().(string); ok && s != "" {
				return s
			}
		}
	}
	return fmt.Sprintf("%T", v)
}
