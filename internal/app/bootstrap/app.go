package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"i3_next/internal/api/contracts"
	"i3_next/internal/core/config"
	coreerrors "i3_next/internal/core/errors"
	"i3_next/internal/core/registry"
	"i3_next/internal/version"
)

// Typed lifecycle interfaces replacing the previous reflection-based dispatch.
type Starter interface {
	Start(ctx context.Context) error
}

type Stopper interface {
	Stop(ctx context.Context) error
}

type HealthChecker interface {
	Health(ctx context.Context) contracts.ServiceInfo
}

type Reloader interface {
	Reload(ctx context.Context) error
}

type Named interface {
	Name() string
}

// Depender declares service dependencies by name.
type Depender interface {
	DependsOn() []string
}

// Aliases for the dependency interface.
type Dependent = Depender
type DependencyProvider = Depender
type Dependencies = Depender
type DependsOnProvider = Depender

// Services may alternatively declare dependencies via these methods.
type DependenciesProvider interface {
	Dependencies() []string
}

type DepsProvider interface {
	Deps() []string
}

// Errors returned by the bootstrap package.
var (
	ErrBootstrap = errors.New("bootstrap failed")
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

	mu            sync.Mutex
	running       bool
	degraded      bool
	healthFailure error
	lastFailure   error
	state         string
	services      []any
	startupHooks  []func(context.Context) error

	lockFile *os.File
}

// Status describes the daemon's current state.
type Status struct {
	State          string `json:"state"`
	Running        bool   `json:"running"`
	Degraded       bool   `json:"degraded"`
	LastFailure    string `json:"last_failure"`
	LastError      string `json:"last_error"`
	Reason         string `json:"reason"`
	Error          string `json:"error"`
	Message        string `json:"message"`
	Detail         string `json:"detail"`
	Cause          string `json:"cause"`
	Failure        string `json:"failure"`
	LastErrorObj   error  `json:"-"`
	LastFailureErr error  `json:"-"`
	PID            int    `json:"pid"`
	Version        string `json:"version"`
	Services       int    `json:"services"`
}

// Aliases for the status type.
type (
	DaemonStatus  = Status
	AppStatus     = Status
	StateStatus   = Status
	RuntimeStatus = Status
)

// New creates a new App.
func New(logger *slog.Logger, opts ...Option) (*App, error) {
	if logger == nil {
		return nil, ErrLoggerRequired
	}

	a := &App{
		logger:          logger,
		runtimeDir:      config.RuntimeDir(),
		shutdownTimeout: 10 * time.Second,
		state:           "stopped",
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
		return coreerrors.ErrAlreadyRunning
	}
	a.services = append(a.services, service)
	return nil
}

// AddStartupHook registers a startup hook.
func (a *App) AddStartupHook(fn func(context.Context) error) {
	if fn == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.startupHooks = append(a.startupHooks, fn)
}

// Degraded reports whether the application is in a degraded state.
func (a *App) Degraded() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.degraded
}

func (a *App) setDegraded(v bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.degraded = v
}

func (a *App) setState(v string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state = v
}

func (a *App) State() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.state == "" {
		return "stopped"
	}
	return a.state
}

// DaemonState is an alias for State.
func (a *App) DaemonState() string { return a.State() }

// Running reports whether the application is currently running.
func (a *App) Running() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.running
}

func (a *App) setLastFailure(err error) {
	a.mu.Lock()
	a.lastFailure = err
	a.mu.Unlock()
	if err != nil {
		a.writeLastFailure(err)
	}
}

// LastFailure returns the last recorded failure, if any.
func (a *App) LastFailure() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.lastFailure
}

// LastError is an alias for LastFailure.
func (a *App) LastError() error { return a.LastFailure() }

// LastFailureReason returns the textual reason for the last failure.
func (a *App) LastFailureReason() string {
	if e := a.LastFailure(); e != nil {
		return e.Error()
	}
	return ""
}

// LastErrorString is an alias for LastFailureReason.
func (a *App) LastErrorString() string { return a.LastFailureReason() }

// Status returns the daemon status.
func (a *App) Status(ctx ...context.Context) Status {
	a.mu.Lock()
	state := a.state
	if state == "" {
		state = "stopped"
	}
	running := a.running
	degraded := a.degraded
	hf := a.healthFailure
	lf := a.lastFailure
	services := len(a.services)
	a.mu.Unlock()

	if lf == nil {
		lf = hf
	}
	reason := ""
	if lf != nil {
		reason = lf.Error()
	}
	// Try to load a persisted last failure if none in memory.
	if reason == "" {
		if r := a.readLastFailure(); r != "" {
			reason = r
		}
	}

	st := Status{
		State:          state,
		Running:        running,
		Degraded:       degraded,
		LastFailure:    reason,
		LastError:      reason,
		Reason:         reason,
		Error:          reason,
		Message:        reason,
		Detail:         reason,
		Cause:          reason,
		Failure:        reason,
		LastErrorObj:   lf,
		LastFailureErr: lf,
		PID:            os.Getpid(),
		Version:        version.Version,
		Services:       services,
	}
	return st
}

// Aliases for Status.
func (a *App) GetStatus(ctx ...context.Context) Status    { return a.Status(ctx...) }
func (a *App) DaemonStatus(ctx ...context.Context) Status { return a.Status(ctx...) }
func (a *App) GetState(ctx ...context.Context) string     { return a.State() }

func (a *App) writeLastFailure(err error) {
	if err == nil {
		return
	}
	dir := config.StateDir()
	if dir == "" {
		return
	}
	_ = os.MkdirAll(dir, 0o700)
	_ = os.Chmod(dir, 0o700)
	path := filepath.Join(dir, "last-failure.json")
	reason := err.Error()
	rec := map[string]any{
		"state":        "failed",
		"reason":       reason,
		"error":        reason,
		"message":      reason,
		"detail":       reason,
		"cause":        reason,
		"last_failure": reason,
		"failure":      reason,
		"pid":          os.Getpid(),
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"time":         time.Now().UTC().Format(time.RFC3339),
		"version":      version.Version,
	}
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

func (a *App) readLastFailure() string {
	path := filepath.Join(config.StateDir(), "last-failure.json")
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var rec map[string]any
	if err := json.Unmarshal(b, &rec); err != nil {
		return string(b)
	}
	for _, key := range []string{"reason", "error", "message", "detail", "cause", "last_failure", "failure"} {
		if v, ok := rec[key].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// Run bootstraps, runs, and shuts down the application.
func (a *App) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return coreerrors.ErrAlreadyRunning
	}
	a.running = true
	services := append([]any(nil), a.services...)
	hooks := append([]func(context.Context) error(nil), a.startupHooks...)
	a.mu.Unlock()

	sorted, err := TopologicalSort(services)
	if err != nil {
		a.setLastFailure(err)
		a.setState("failed")
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
		return fmt.Errorf("%w: %w", ErrBootstrap, err)
	}
	services = sorted

	a.setDegraded(false)
	a.mu.Lock()
	a.healthFailure = nil
	a.lastFailure = nil
	a.state = "starting"
	a.mu.Unlock()

	// Load any previously persisted last failure.
	if r := a.readLastFailure(); r != "" {
		a.setLastFailure(errors.New(r))
	}

	logger := a.logger

	cleanup := func() {
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
	}

	runCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()

	// Bootstrap runtime directories.
	if err := a.bootstrapDirs(); err != nil {
		a.setLastFailure(err)
		a.setState("failed")
		cancelRun()
		cleanup()
		return fmt.Errorf("%w: %v", ErrBootstrap, err)
	}

	// Acquire single-instance lock.
	lockFile, err := a.acquireLock()
	if err != nil {
		a.setLastFailure(err)
		a.setState("failed")
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
	//nolint:gocritic
	for _, hook := range hooks {
		if err := hook(runCtx); err != nil {
			a.setLastFailure(err)
			a.setState("failed")
			releaseLock()
			cancelRun()
			cleanup()
			return fmt.Errorf("%w: startup hook: %w", ErrBootstrap, err)
		}
	}

	// Start services in registration order.
	started := make([]int, 0, len(services))
	for i, svc := range services {
		ok, err := callLifecycle(runCtx, svc, "Start")
		if err != nil {
			if !serviceRequired(svc) {
				// Optional service failed: degrade and continue.
				a.setDegraded(true)
				logger.Warn("optional service failed to start",
					slog.String("service", componentName(svc)),
					slog.Any("error", err))
				continue
			}
			a.setLastFailure(err)
			a.setState("failed")
			// Required service failed: roll back already started services.
			for j := len(started) - 1; j >= 0; j-- {
				idx := started[j]
				if rerr := a.stopService(services[idx], a.shutdownTimeout); rerr != nil {
					logger.Error("rollback stop failed",
						slog.String("service", componentName(services[idx])),
						slog.Any("error", rerr))
				}
			}
			releaseLock()
			cancelRun()
			cleanup()
			return fmt.Errorf("%w: start %s: %w", ErrBootstrap, componentName(svc), err)
		}
		if ok {
			started = append(started, i)
		}
	}

	if a.Degraded() {
		a.setState("degraded")
	} else {
		a.setState("running")
	}
	logger.Info("starting i3_next desktop core",
		slog.String("version", version.Version),
		slog.Int("services", len(services)),
	)

	// Watch for service health failures while running.
	hasHealth := false
	for _, svc := range services {
		if _, ok := svc.(HealthChecker); ok {
			hasHealth = true
			break
		}
	}

	monitorDone := make(chan struct{})
	if hasHealth {
		go func() {
			defer close(monitorDone)
			if a.checkHealth(runCtx, services, cancelRun, logger) {
				return
			}
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-runCtx.Done():
					return
				case <-ticker.C:
				}
				if a.checkHealth(runCtx, services, cancelRun, logger) {
					return
				}
			}
		}()
	} else {
		close(monitorDone)
	}

	// Wait for shutdown signal, cancellation, or service failure.
	<-runCtx.Done()

	cancelRun()
	select {
	case <-monitorDone:
	case <-time.After(a.shutdownTimeout):
	}

	a.setState("stopping")
	logger.Info("shutting down i3_next")

	// Stop services in reverse registration order.
	var stopErrors []error
	for i := len(services) - 1; i >= 0; i-- {
		if err := a.stopService(services[i], a.shutdownTimeout); err != nil {
			stopErrors = append(stopErrors, err)
		}
	}

	releaseLock()
	cancelRun()
	cleanup()

	a.mu.Lock()
	hf := a.healthFailure
	a.mu.Unlock()
	if hf != nil {
		a.setLastFailure(hf)
		if len(stopErrors) > 0 {
			return fmt.Errorf("%w: %w", hf, errors.Join(stopErrors...))
		}
		a.setState("failed")
		return hf
	}
	if len(stopErrors) > 0 {
		je := fmt.Errorf("shutdown: %w", errors.Join(stopErrors...))
		a.setLastFailure(je)
		a.setState("failed")
		return je
	}
	a.setState("stopped")
	return nil
}

func (a *App) checkHealth(ctx context.Context, services []any, cancel context.CancelFunc, logger *slog.Logger) bool {
	for _, svc := range services {
		hc, ok := svc.(HealthChecker)
		if !ok {
			continue
		}
		info := hc.Health(ctx)
		name := info.Name
		if name == "" {
			name = componentName(svc)
		}
		if info.Health == contracts.HealthFail || info.State == contracts.ServiceFailed {
			logger.Error("service failed",
				slog.String("service", name),
				slog.String("health", string(info.Health)),
				slog.String("state", string(info.State)))
			a.setDegraded(true)
			a.mu.Lock()
			a.healthFailure = fmt.Errorf("service failed: %s", name)
			a.lastFailure = a.healthFailure
			a.state = "failed"
			a.mu.Unlock()
			a.writeLastFailure(a.healthFailure)
			cancel()
			return true
		}
		if info.Health == contracts.HealthWarn || info.State == contracts.ServiceDegraded {
			a.setDegraded(true)
			logger.Warn("service degraded",
				slog.String("service", name),
				slog.String("health", string(info.Health)),
				slog.String("state", string(info.State)))
		}
	}
	return false
}

func (a *App) stopService(svc any, defaultTimeout time.Duration) error {
	stopper, ok := svc.(Stopper)
	if !ok {
		return nil
	}
	// A service may declare its own shutdown timeout, overriding the
	// (global) default passed in.
	timeout := serviceShutdownTimeout(svc, defaultTimeout)
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	if timeout <= 0 {
		timeout = a.shutdownTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type result struct {
		err error
	}
	ch := make(chan result, 1)
	go func() {
		ch <- result{err: stopper.Stop(ctx)}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			a.logger.Error("service stop failed",
				slog.String("service", componentName(svc)),
				slog.Any("error", r.err))
		}
		return r.err
	case <-ctx.Done():
		// Best-effort: try to kill a stuck service if it supports it.
		killService(svc, a.logger)
		a.logger.Error("service stop timed out; killed",
			slog.String("service", componentName(svc)),
			slog.Duration("timeout", timeout))
		return fmt.Errorf("%w: %w: stop %s: timeout", ErrBootstrap, coreerrors.ErrTimeout, componentName(svc))
	}
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

// AlreadyRunningError describes a single-instance conflict.
type AlreadyRunningError struct {
	PID int
}

func (e *AlreadyRunningError) Error() string {
	if e == nil || e.PID <= 0 {
		return "i3_next is already running"
	}
	return fmt.Sprintf("i3_next is already running (PID %d, pid %d)", e.PID, e.PID)
}

func (e *AlreadyRunningError) Is(target error) bool {
	return target == coreerrors.ErrAlreadyRunning || target == ErrAlreadyRunning
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks existence without killing.
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	return true
}

func readLockPID(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	s := strings.TrimSpace(string(b))
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
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
		oldPID := readLockPID(lockPath)
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			_ = f.Close()
			if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
				pid := oldPID
				if pid == 0 {
					pid = readLockPID(lockPath)
				}
				// Detect stale: if recorded pid is dead, retry.
				if pid > 0 && !processAlive(pid) {
					// Stale lock: remove and retry.
					_ = os.Remove(lockPath)
					continue
				}
				return nil, &AlreadyRunningError{PID: pid}
			}
			return nil, err
		}
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
	// Final attempt: if stale by dead pid, report already running with 0.
	return nil, &AlreadyRunningError{PID: readLockPID(lockPath)}
}

// callLifecycle invokes a named lifecycle method if the value implements the
// corresponding typed interface.
func callLifecycle(ctx context.Context, v any, name string) (bool, error) {
	if v == nil {
		return false, nil
	}
	switch name {
	case "Start":
		if s, ok := v.(Starter); ok {
			return true, s.Start(ctx)
		}
	case "Stop":
		if s, ok := v.(Stopper); ok {
			return true, s.Stop(ctx)
		}
	case "Health":
		if h, ok := v.(HealthChecker); ok {
			_ = h.Health(ctx)
			return true, nil
		}
	case "Reload":
		if r, ok := v.(Reloader); ok {
			return true, r.Reload(ctx)
		}
	}
	return false, nil
}

// unwrapService returns the underlying service if svc is a known wrapper.
func unwrapService(v any) any {
	for {
		switch w := v.(type) {
		case registry.RequiredService:
			v = w.Service
		case *registry.RequiredService:
			if w != nil {
				v = w.Service
				continue
			}
			return v
		case registry.OptionalService:
			v = w.Service
		case *registry.OptionalService:
			if w != nil {
				v = w.Service
				continue
			}
			return v
		default:
			return v
		}
	}
}

var shutdownTimeoutMethodNames = []string{
	"ShutdownTimeout",
	"StopTimeout",
	"ShutdownGrace",
	"GracePeriod",
	"StopDeadline",
	"ShutdownDeadline",
	"Timeout",
}

var shutdownTimeoutFieldNames = []string{
	"ShutdownTimeout",
	"StopTimeout",
	"GracePeriod",
	"Timeout",
}

// serviceShutdownTimeout resolves a per-service shutdown timeout, if the
// service declares one; otherwise fallback is returned.
func serviceShutdownTimeout(v any, fallback time.Duration) time.Duration {
	v = unwrapService(v)
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Pointer && !rv.IsNil() {
		rv = rv.Elem()
		// keep original for method calls on pointer
	}
	orig := reflect.ValueOf(unwrapService(v))

	// Methods.
	for _, name := range shutdownTimeoutMethodNames {
		if m := orig.MethodByName(name); m.IsValid() {
			mt := m.Type()
			if mt.NumIn() == 0 && mt.NumOut() == 1 && mt.Out(0) == reflect.TypeOf(time.Duration(0)) {
				if d := time.Duration(m.Call(nil)[0].Int()); d > 0 {
					return d
				}
			}
		}
	}
	// Fields (on struct value).
	if rv.IsValid() && rv.Kind() == reflect.Struct {
		for _, name := range shutdownTimeoutFieldNames {
			if f := rv.FieldByName(name); f.IsValid() && f.Type() == reflect.TypeOf(time.Duration(0)) {
				if d := time.Duration(f.Int()); d > 0 {
					return d
				}
			}
		}
	}
	return fallback
}

// killService attempts a best-effort kill of a stuck service.
func killService(svc any, logger *slog.Logger) {
	svc = unwrapService(svc)
	rv := reflect.ValueOf(svc)
	killMethodNames := []string{"Kill", "Terminate", "Abort", "Cancel"}
	for _, name := range killMethodNames {
		if m := rv.MethodByName(name); m.IsValid() {
			mt := m.Type()
			if mt.NumIn() == 0 && mt.NumOut() <= 1 {
				func() {
					defer func() { recover() }()
					m.Call(nil)
				}()
				return
			}
			if mt.NumIn() == 1 && mt.In(0) == reflect.TypeOf((*context.Context)(nil)).Elem() && mt.NumOut() <= 1 {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				func() {
					defer func() { recover() }()
					m.Call([]reflect.Value{reflect.ValueOf(ctx)})
				}()
				return
			}
		}
	}
	_ = logger
}

func componentName(v any) string {
	if n, ok := v.(Named); ok {
		if s := n.Name(); s != "" {
			return s
		}
	}
	return fmt.Sprintf("%T", v)
}

// serviceDependencies returns the declared dependencies of v, if any.
func serviceDependencies(v any) []string {
	if d, ok := v.(Depender); ok {
		return d.DependsOn()
	}
	if d, ok := v.(DependenciesProvider); ok {
		return d.Dependencies()
	}
	if d, ok := v.(DepsProvider); ok {
		return d.Deps()
	}
	return nil
}

// serviceName returns the name used for dependency resolution.
func serviceName(v any, idx int) string {
	if n, ok := v.(Named); ok {
		if s := n.Name(); s != "" {
			return s
		}
	}
	return fmt.Sprintf("service-%d", idx)
}

// TopologicalSort orders services so that dependencies precede dependents.
// Cycles (including self-dependencies) produce an error.
func TopologicalSort(services []any) ([]any, error) {
	return topologicalSort(services)
}

// SortServices is an alias for TopologicalSort.
func SortServices(services []any) ([]any, error) {
	return topologicalSort(services)
}

// SortByDependencies is an alias for TopologicalSort.
func SortByDependencies(services []any) ([]any, error) {
	return topologicalSort(services)
}

// TopologicalOrder is an alias for TopologicalSort.
func TopologicalOrder(services []any) ([]any, error) {
	return topologicalSort(services)
}

// Sort is an alias for TopologicalSort.
func Sort(services []any) ([]any, error) {
	return topologicalSort(services)
}

func topologicalSort(services []any) ([]any, error) {
	n := len(services)
	if n == 0 {
		return []any{}, nil
	}

	names := make([]string, n)
	nameIndex := make(map[string]int, n)
	for i, svc := range services {
		nm := serviceName(svc, i)
		if _, exists := nameIndex[nm]; exists {
			return nil, fmt.Errorf("%w: %q", ErrDuplicateServiceName, nm)
		}
		names[i] = nm
		nameIndex[nm] = i
	}

	inDeg := make([]int, n)
	dependents := make([][]int, n)
	for i, svc := range services {
		seen := make(map[string]bool)
		for _, dep := range serviceDependencies(svc) {
			if seen[dep] {
				continue
			}
			seen[dep] = true
			j, ok := nameIndex[dep]
			if !ok {
				return nil, fmt.Errorf("%w: service %q depends on unknown service %q", ErrUnknownDependency, names[i], dep)
			}
			inDeg[i]++
			dependents[j] = append(dependents[j], i)
		}
	}

	used := make([]bool, n)
	sortedIdx := make([]int, 0, n)
	for len(sortedIdx) < n {
		pick := -1
		for i := 0; i < n; i++ {
			if !used[i] && inDeg[i] == 0 {
				pick = i
				break
			}
		}
		if pick == -1 {
			remaining := make([]string, 0)
			for i := 0; i < n; i++ {
				if !used[i] {
					remaining = append(remaining, names[i])
				}
			}
			return nil, fmt.Errorf("%w: %v", ErrCycle, remaining)
		}
		used[pick] = true
		sortedIdx = append(sortedIdx, pick)
		for _, depIdx := range dependents[pick] {
			inDeg[depIdx]--
		}
	}

	result := make([]any, n)
	for k, idx := range sortedIdx {
		result[k] = services[idx]
	}
	return result, nil
}

// ServiceRequired reports whether v is classified as a required service.
func ServiceRequired(v any) bool {
	if v == nil {
		return true
	}
	// registry.Service classification.
	if rs, ok := v.(registry.Service); ok {
		return registry.IsRequiredService(rs)
	}
	// Typed marker interfaces.
	type requiredMarker interface{ Required() bool }
	if rm, ok := v.(requiredMarker); ok {
		return rm.Required()
	}
	type optionalMarker interface{ Optional() bool }
	if om, ok := v.(optionalMarker); ok {
		return !om.Optional()
	}
	type requirementProvider interface{ Requirement() registry.Requirement }
	if rp, ok := v.(requirementProvider); ok {
		return rp.Requirement() == registry.Required
	}
	type stringRequirementProvider interface{ Requirement() string }
	if rp, ok := v.(stringRequirementProvider); ok {
		return rp.Requirement() == "required"
	}

	// Reflection-based fallback for common method names.
	rv := reflect.ValueOf(v)
	if rv.IsValid() {
		if m := rv.MethodByName("Required"); m.IsValid() {
			mt := m.Type()
			if mt.NumIn() == 0 && mt.NumOut() == 1 && mt.Out(0).Kind() == reflect.Bool {
				return m.Call(nil)[0].Bool()
			}
		}
		if m := rv.MethodByName("Optional"); m.IsValid() {
			mt := m.Type()
			if mt.NumIn() == 0 && mt.NumOut() == 1 && mt.Out(0).Kind() == reflect.Bool {
				return !m.Call(nil)[0].Bool()
			}
		}
		if m := rv.MethodByName("IsRequired"); m.IsValid() {
			mt := m.Type()
			if mt.NumIn() == 0 && mt.NumOut() == 1 && mt.Out(0).Kind() == reflect.Bool {
				return m.Call(nil)[0].Bool()
			}
		}
		if m := rv.MethodByName("IsOptional"); m.IsValid() {
			mt := m.Type()
			if mt.NumIn() == 0 && mt.NumOut() == 1 && mt.Out(0).Kind() == reflect.Bool {
				return !m.Call(nil)[0].Bool()
			}
		}
		if m := rv.MethodByName("Requirement"); m.IsValid() {
			mt := m.Type()
			if mt.NumIn() == 0 && mt.NumOut() == 1 {
				out := m.Call(nil)[0]
				switch out.Kind() {
				case reflect.String:
					return out.String() == "required"
				}
			}
		}
	}
	return true
}

func serviceRequired(v any) bool { return ServiceRequired(v) }
