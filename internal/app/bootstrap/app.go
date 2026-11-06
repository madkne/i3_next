package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"i3_next/internal/api/contracts"
	"i3_next/internal/core/config"
	coreerrors "i3_next/internal/core/errors"
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
	services      []any
	startupHooks  []func(context.Context) error

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
		a.mu.Lock()
		a.running = false
		a.mu.Unlock()
		return fmt.Errorf("%w: %w", ErrBootstrap, err)
	}
	services = sorted

	a.setDegraded(false)
	a.mu.Lock()
	a.healthFailure = nil
	a.mu.Unlock()

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
		cancelRun()
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
		if err := hook(runCtx); err != nil {
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
			// Roll back already started services.
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
		if len(stopErrors) > 0 {
			return fmt.Errorf("%w: %w", hf, errors.Join(stopErrors...))
		}
		return hf
	}
	if len(stopErrors) > 0 {
		return fmt.Errorf("shutdown: %w", errors.Join(stopErrors...))
	}
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
			a.mu.Unlock()
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

func (a *App) stopService(svc any, timeout time.Duration) error {
	stopper, ok := svc.(Stopper)
	if !ok {
		return nil
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
		a.logger.Error("service stop timed out",
			slog.String("service", componentName(svc)))
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
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			_ = f.Close()
			if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
				return nil, coreerrors.ErrAlreadyRunning
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
	return nil, coreerrors.ErrAlreadyRunning
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
