package supervisor

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"i3_next/internal/api/contracts"
)

// ProcessState represents the lifecycle state of a supervised process.
type ProcessState string

const (
	StatePending    ProcessState = "pending"
	StateStarting   ProcessState = "starting"
	StateRunning    ProcessState = "running"
	StateRestarting ProcessState = "restarting"
	StateStopping   ProcessState = "stopping"
	StateStopped    ProcessState = "stopped"
	StateExited     ProcessState = "exited"
	StateFailed     ProcessState = "failed"
	StateDegraded   ProcessState = "degraded"
)

// RestartPolicy configures automatic restarts.
type RestartPolicy struct {
	Enabled bool

	MaxRestarts     int
	MaxRestart      int
	RestartLimit    int
	MaxRestartCount int
	Restarts        int

	Window         time.Duration
	RestartWindow  time.Duration
	WindowDuration time.Duration

	InitialBackoff time.Duration
	InitialDelay   time.Duration
	Backoff        time.Duration
	Delay          time.Duration

	MaxBackoff time.Duration
	MaxDelay   time.Duration

	Multiplier    float64
	BackoffFactor float64
	Factor        float64

	Jitter        float64
	JitterEnabled bool
}

// ProcessSpec describes a child process.
type ProcessSpec struct {
	Name  string
	ID    string
	Label string
	Title string

	Command    string
	Path       string
	Executable string
	Binary     string
	Program    string

	Args      []string
	Argv      []string
	Arguments []string

	Env         map[string]string
	Environment map[string]string
	EnvVars     map[string]string
	EnvList     []string

	EnvAllowlist []string
	AllowedEnv   []string
	EnvAllow     []string
	AllowEnv     []string

	Dir        string
	WorkingDir string
	Cwd        string

	GracePeriod     time.Duration
	StopTimeout     time.Duration
	ShutdownTimeout time.Duration
	Timeout         time.Duration

	Restart       RestartPolicy
	RestartPolicy RestartPolicy
	Policy        RestartPolicy

	MaxRestarts int
	Window      time.Duration

	Logger *slog.Logger

	Stdout io.Writer
	Stderr io.Writer

	Supervisor *Supervisor
}

// ProcessStatus is a snapshot of process state.
type ProcessStatus struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	PID       int    `json:"pid"`
	ExitCode  int    `json:"exit_code"`
	Restarts  int    `json:"restarts"`
	LastError string `json:"last_error,omitempty"`
	Degraded  bool   `json:"degraded"`
}

// Aliases.
type (
	Child             = Process
	ManagedProcess    = Process
	SupervisedProcess = Process
	Job               = Process
	Task              = Process
	ProcessConfig     = ProcessSpec
	Config            = ProcessSpec
	Spec              = ProcessSpec
	ProcessInfo       = ProcessStatus
	Status            = ProcessStatus
)

// Supervisor manages child processes.
type Supervisor struct {
	logger *slog.Logger

	mu        sync.Mutex
	processes map[string]*Process

	gracePeriod   time.Duration
	envAllow      []string
	defaultPolicy RestartPolicy

	redactPatterns []string
}

// Option configures a Supervisor.
type Option func(*Supervisor)

func WithLogger(l *slog.Logger) Option {
	return func(s *Supervisor) {
		if l != nil {
			s.logger = l
		}
	}
}

func WithGracePeriod(d time.Duration) Option {
	return func(s *Supervisor) {
		if d > 0 {
			s.gracePeriod = d
		}
	}
}

func WithStopTimeout(d time.Duration) Option     { return WithGracePeriod(d) }
func WithShutdownTimeout(d time.Duration) Option { return WithGracePeriod(d) }

func WithEnvAllowlist(list []string) Option {
	return func(s *Supervisor) {
		if len(list) > 0 {
			s.envAllow = append([]string(nil), list...)
		}
	}
}

func WithAllowedEnv(list []string) Option { return WithEnvAllowlist(list) }
func WithEnvAllow(list []string) Option   { return WithEnvAllowlist(list) }

//nolint:gocritic // public API takes value
func WithRestartPolicy(p RestartPolicy) Option {
	return func(s *Supervisor) {
		s.defaultPolicy = mergePolicy(s.defaultPolicy, p)
	}
}
func WithMaxRestarts(n int) Option {
	return func(s *Supervisor) {
		s.defaultPolicy.MaxRestarts = n
		s.defaultPolicy.Enabled = true
	}
}
func WithRestartWindow(d time.Duration) Option {
	return func(s *Supervisor) {
		s.defaultPolicy.Window = d
	}
}
func WithInitialBackoff(d time.Duration) Option {
	return func(s *Supervisor) {
		s.defaultPolicy.InitialBackoff = d
	}
}
func WithMaxBackoff(d time.Duration) Option {
	return func(s *Supervisor) {
		s.defaultPolicy.MaxBackoff = d
	}
}
func WithBackoffMultiplier(f float64) Option {
	return func(s *Supervisor) {
		s.defaultPolicy.Multiplier = f
	}
}

func New(logger *slog.Logger, opts ...Option) *Supervisor {
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	s := &Supervisor{
		logger:      logger,
		processes:   make(map[string]*Process),
		gracePeriod: 5 * time.Second,
		envAllow: []string{
			"PATH", "HOME", "USER", "LOGNAME", "SHELL", "LANG", "LC_*",
			"TERM", "DISPLAY", "WAYLAND_DISPLAY", "SWAYSOCK", "XDG_*",
			"XAUTHORITY", "TMPDIR", "TZ",
		},
		redactPatterns: []string{"TOKEN", "SECRET", "PASSWORD"},
	}
	for _, o := range opts {
		if o != nil {
			o(s)
		}
	}
	return s
}

func NewSupervisor(logger *slog.Logger, opts ...Option) *Supervisor { return New(logger, opts...) }
func NewProcessSupervisor(logger *slog.Logger, opts ...Option) *Supervisor {
	return New(logger, opts...)
}

// Aliases for constructor name.
var NewManager = NewSupervisor

// Process is a supervised child process.
type Process struct {
	supervisor *Supervisor
	logger     *slog.Logger

	mu sync.Mutex

	name string
	spec ProcessSpec
	pol  RestartPolicy

	cmd    *exec.Cmd
	pid    int
	pgid   int
	cancel context.CancelFunc
	done   chan struct{}

	state ProcessState

	exitCode int
	signal   os.Signal
	lastErr  error

	restarts []time.Time
	backoff  time.Duration

	stopRequested bool
	stopped       chan struct{}
}

func NewProcess(args ...any) *Process {
	s := New(nil)
	spec := s.parseSpec(args...)
	return s.newProcess(spec)
}

//nolint:gocritic // internal copy
func (s *Supervisor) newProcess(spec ProcessSpec) *Process {
	if spec.Logger == nil {
		spec.Logger = s.logger
	}
	if spec.Name == "" {
		spec.Name = spec.Command
		if spec.Name == "" {
			spec.Name = "process"
		}
	}
	spec.Name = filepath.Base(spec.Name)
	if spec.Name == "" || spec.Name == "." || spec.Name == "/" {
		spec.Name = "process"
	}
	pol := mergePolicy(spec.Restart, s.defaultPolicy)
	p := &Process{
		supervisor: s,
		logger:     spec.Logger,
		name:       spec.Name,
		spec:       spec,
		pol:        pol,
		state:      StatePending,
		stopped:    make(chan struct{}),
	}
	// merge top-level policy fields
	if spec.RestartPolicy.MaxRestarts != 0 || spec.RestartPolicy.Window != 0 {
		p.pol = mergePolicy(p.pol, spec.RestartPolicy)
	}
	if spec.Policy.MaxRestarts != 0 || spec.Policy.Window != 0 {
		p.pol = mergePolicy(p.pol, spec.Policy)
	}
	// top-level shortcuts
	if spec.MaxRestarts != 0 {
		p.pol.MaxRestarts = spec.MaxRestarts
	}
	if spec.Window != 0 {
		p.pol.Window = spec.Window
	}
	return p
}

//nolint:gocritic // value semantics
func mergePolicy(a, b RestartPolicy) RestartPolicy {
	if b.Enabled {
		a.Enabled = true
	}
	if b.MaxRestarts != 0 {
		a.MaxRestarts = b.MaxRestarts
	}
	if b.MaxRestart != 0 {
		a.MaxRestarts = b.MaxRestart
	}
	if b.RestartLimit != 0 {
		a.MaxRestarts = b.RestartLimit
	}
	if b.MaxRestartCount != 0 {
		a.MaxRestarts = b.MaxRestartCount
	}
	if b.Restarts != 0 {
		a.MaxRestarts = b.Restarts
	}
	if b.Window != 0 {
		a.Window = b.Window
	}
	if b.RestartWindow != 0 {
		a.Window = b.RestartWindow
	}
	if b.WindowDuration != 0 {
		a.Window = b.WindowDuration
	}
	if b.InitialBackoff != 0 {
		a.InitialBackoff = b.InitialBackoff
	}
	if b.InitialDelay != 0 {
		a.InitialBackoff = b.InitialDelay
	}
	if b.Backoff != 0 {
		a.InitialBackoff = b.Backoff
	}
	if b.Delay != 0 {
		a.InitialBackoff = b.Delay
	}
	if b.MaxBackoff != 0 {
		a.MaxBackoff = b.MaxBackoff
	}
	if b.MaxDelay != 0 {
		a.MaxBackoff = b.MaxDelay
	}
	if b.Multiplier != 0 {
		a.Multiplier = b.Multiplier
	}
	if b.BackoffFactor != 0 {
		a.Multiplier = b.BackoffFactor
	}
	if b.Factor != 0 {
		a.Multiplier = b.Factor
	}
	if b.Jitter != 0 {
		a.Jitter = b.Jitter
	}
	if b.JitterEnabled {
		a.JitterEnabled = true
		if a.Jitter == 0 {
			a.Jitter = 0.2
		}
	}
	if b.Enabled {
		a.Enabled = true
	}
	return a
}

func (s *Supervisor) parseSpec(args ...any) ProcessSpec {
	spec := ProcessSpec{}
	for _, a := range args {
		switch v := a.(type) {
		case ProcessSpec:
			spec = mergeSpec(spec, v)
		case *ProcessSpec:
			if v != nil {
				spec = mergeSpec(spec, *v)
			}
		case *Process:
			if v != nil {
				spec.Name = v.Name()
				spec.Restart = v.pol
			}
		case string:
			if spec.Name == "" { //nolint:gocritic
				spec.Name = v
			} else if spec.Command == "" {
				spec.Command = v
			} else {
				spec.Args = append(spec.Args, v)
			}
		case []string:
			spec.Args = append(spec.Args, v...)
		case []any:
			for _, e := range v {
				if s, ok := e.(string); ok {
					spec.Args = append(spec.Args, s)
				}
			}
		case map[string]string:
			if spec.Env == nil {
				spec.Env = make(map[string]string)
			}
			for k, val := range v {
				spec.Env[k] = val
			}
		case RestartPolicy:
			spec.Restart = mergePolicy(spec.Restart, v)
		case *RestartPolicy:
			if v != nil {
				spec.Restart = mergePolicy(spec.Restart, *v)
			}
		case time.Duration:
			if spec.GracePeriod == 0 {
				spec.GracePeriod = v
			} else if spec.StopTimeout == 0 {
				spec.StopTimeout = v
			}
		case *slog.Logger:
			spec.Logger = v
		case Option:
			// ignore
		default:
			// reflection-based field extraction
			// reflection-based extraction not supported
			_ = a
		}
	}
	// normalize command
	if spec.Command == "" {
		spec.Command = spec.Path
	}
	if spec.Command == "" {
		spec.Command = spec.Executable
	}
	if spec.Command == "" {
		spec.Command = spec.Binary
	}
	if spec.Command == "" {
		spec.Command = spec.Program
	}
	if spec.Args == nil {
		if spec.Argv != nil {
			spec.Args = spec.Argv
		} else if spec.Arguments != nil {
			spec.Args = spec.Arguments
		}
	} else {
		if spec.Argv != nil {
			spec.Args = append([]string{}, spec.Argv...)
		}
	}
	if spec.Dir == "" {
		spec.Dir = spec.WorkingDir
	}
	if spec.Dir == "" {
		spec.Dir = spec.Cwd
	}
	if spec.Env == nil {
		if spec.Environment != nil {
			spec.Env = spec.Environment
		} else if spec.EnvVars != nil {
			spec.Env = spec.EnvVars
		}
	}
	if spec.EnvAllowlist == nil {
		if spec.AllowedEnv != nil { //nolint:gocritic
			spec.EnvAllowlist = spec.AllowedEnv
		} else if spec.EnvAllow != nil {
			spec.EnvAllowlist = spec.EnvAllow
		} else if spec.AllowEnv != nil {
			spec.EnvAllowlist = spec.AllowEnv
		}
	}
	if spec.GracePeriod == 0 {
		spec.GracePeriod = spec.StopTimeout
	}
	if spec.GracePeriod == 0 {
		spec.GracePeriod = spec.ShutdownTimeout
	}
	if spec.GracePeriod == 0 {
		spec.GracePeriod = spec.Timeout
	}
	if spec.Logger == nil {
		spec.Logger = s.logger
	}
	if spec.Supervisor == nil {
		spec.Supervisor = s
	}
	// merge top-level restart shortcuts
	if spec.MaxRestarts != 0 {
		spec.Restart.MaxRestarts = spec.MaxRestarts
	}
	if spec.Window != 0 {
		spec.Restart.Window = spec.Window
	}
	return spec
}

//nolint:gocritic // value semantics
func mergeSpec(a, b ProcessSpec) ProcessSpec {
	if b.Name != "" {
		a.Name = b.Name
	}
	if b.ID != "" {
		a.ID = b.ID
	}
	if b.Label != "" {
		a.Label = b.Label
	}
	if b.Title != "" {
		a.Title = b.Title
	}
	if b.Command != "" {
		a.Command = b.Command
	}
	if b.Path != "" {
		a.Path = b.Path
	}
	if b.Executable != "" {
		a.Executable = b.Executable
	}
	if b.Binary != "" {
		a.Binary = b.Binary
	}
	if b.Program != "" {
		a.Program = b.Program
	}
	if len(b.Args) > 0 {
		a.Args = append([]string{}, b.Args...)
	}
	if len(b.Argv) > 0 {
		a.Argv = append([]string{}, b.Argv...)
	}
	if len(b.Arguments) > 0 {
		a.Arguments = append([]string{}, b.Arguments...)
	}
	if b.Env != nil {
		if a.Env == nil {
			a.Env = make(map[string]string)
		}
		for k, v := range b.Env {
			a.Env[k] = v
		}
	}
	if b.Environment != nil {
		if a.Env == nil {
			a.Env = make(map[string]string)
		}
		for k, v := range b.Environment {
			a.Env[k] = v
		}
	}
	if b.EnvVars != nil {
		if a.Env == nil {
			a.Env = make(map[string]string)
		}
		for k, v := range b.EnvVars {
			a.Env[k] = v
		}
	}
	if len(b.EnvList) > 0 {
		a.EnvList = append([]string{}, b.EnvList...)
	}
	if len(b.EnvAllowlist) > 0 {
		a.EnvAllowlist = append([]string{}, b.EnvAllowlist...)
	}
	if len(b.AllowedEnv) > 0 {
		a.EnvAllowlist = append([]string{}, b.AllowedEnv...)
	}
	if len(b.EnvAllow) > 0 {
		a.EnvAllowlist = append([]string{}, b.EnvAllow...)
	}
	if len(b.AllowEnv) > 0 {
		a.EnvAllowlist = append([]string{}, b.AllowEnv...)
	}
	if b.Dir != "" {
		a.Dir = b.Dir
	}
	if b.WorkingDir != "" {
		a.WorkingDir = b.WorkingDir
	}
	if b.Cwd != "" {
		a.Cwd = b.Cwd
	}
	if b.GracePeriod != 0 {
		a.GracePeriod = b.GracePeriod
	}
	if b.StopTimeout != 0 {
		a.StopTimeout = b.StopTimeout
	}
	if b.ShutdownTimeout != 0 {
		a.ShutdownTimeout = b.ShutdownTimeout
	}
	if b.Timeout != 0 {
		a.Timeout = b.Timeout
	}
	a.Restart = mergePolicy(a.Restart, b.Restart)
	a.RestartPolicy = mergePolicy(a.RestartPolicy, b.RestartPolicy)
	a.Policy = mergePolicy(a.Policy, b.Policy)
	if b.MaxRestarts != 0 {
		a.MaxRestarts = b.MaxRestarts
	}
	if b.Window != 0 {
		a.Window = b.Window
	}
	if b.Logger != nil {
		a.Logger = b.Logger
	}
	if b.Stdout != nil {
		a.Stdout = b.Stdout
	}
	if b.Stderr != nil {
		a.Stderr = b.Stderr
	}
	if b.Supervisor != nil {
		a.Supervisor = b.Supervisor
	}
	return a
}

// Name returns the process name.
func (p *Process) Name() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.name != "" {
		return p.name
	}
	return p.spec.Name
}

// State returns the current state string.
func (p *Process) State() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.state == "" {
		return string(StatePending)
	}
	return string(p.state)
}

func (p *Process) setState(s ProcessState) {
	p.mu.Lock()
	p.state = s
	p.mu.Unlock()
}

// PID returns the OS pid.
func (p *Process) PID() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.pid
}

// ExitCode returns the last exit code.
func (p *Process) ExitCode() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.exitCode
}

// RestartCount returns number of restarts.
func (p *Process) RestartCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.restarts)
}

// Restarts returns number of restarts (alias).
func (p *Process) Restarts() int { return p.RestartCount() }

// RestartsList returns timestamps.
func (p *Process) RestartsList() []time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]time.Time, len(p.restarts))
	copy(out, p.restarts)
	return out
}

// Done returns a channel closed when process supervision ends.
func (p *Process) Done() <-chan struct{} {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.done == nil {
		p.done = make(chan struct{})
	}
	return p.done
}

// LastError returns the last error.
func (p *Process) LastError() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastErr
}

// Status returns a snapshot.
func (p *Process) Status() ProcessStatus {
	p.mu.Lock()
	defer p.mu.Unlock()
	errStr := ""
	if p.lastErr != nil {
		errStr = p.lastErr.Error()
	}
	return ProcessStatus{
		Name:      p.name,
		State:     string(p.state),
		PID:       p.pid,
		ExitCode:  p.exitCode,
		Restarts:  len(p.restarts),
		LastError: errStr,
		Degraded:  p.state == StateDegraded,
	}
}

// Aliases for Status method.
func (p *Process) GetStatus() ProcessStatus { return p.Status() }
func (p *Process) Info() ProcessStatus      { return p.Status() }

// Start begins supervision.
func (p *Process) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	p.mu.Lock()
	if p.cancel != nil {
		p.mu.Unlock()
		return errors.New("process already running")
	}
	runCtx, cancel := context.WithCancel(ctx)
	p.cancel = cancel
	p.done = make(chan struct{})
	p.stopRequested = false
	p.stopped = make(chan struct{})
	p.mu.Unlock()

	go p.runLoop(runCtx)
	return nil
}

// Stop stops the process.
func (p *Process) Stop(ctx context.Context) error {
	p.mu.Lock()
	if p.stopRequested {
		// already stopping
	} else {
		p.stopRequested = true
	}
	cancel := p.cancel
	pid := p.pid
	grace := p.gracePeriod()
	stopped := p.stopped
	p.state = StateStopping
	p.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	if pid > 0 {
		_ = syscall.Kill(-pid, syscall.SIGTERM)
	}

	timer := time.NewTimer(grace)
	defer timer.Stop()
	select {
	case <-stopped:
		return nil
	case <-timer.C:
	}
	if pid > 0 {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}
	select {
	case <-stopped:
	case <-time.After(grace):
	}
	return nil
}

func (p *Process) gracePeriod() time.Duration {
	if p.spec.GracePeriod > 0 {
		return p.spec.GracePeriod
	}
	if p.supervisor != nil && p.supervisor.gracePeriod > 0 {
		return p.supervisor.gracePeriod
	}
	return 5 * time.Second
}

// Wait blocks until supervision ends.
func (p *Process) Wait() error {
	<-p.Done()
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.lastErr
}

func (p *Process) WaitContext(ctx context.Context) error {
	select {
	case <-p.Done():
		p.mu.Lock()
		defer p.mu.Unlock()
		return p.lastErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Restart restarts the process.
func (p *Process) Restart(ctx context.Context) error {
	_ = p.Stop(ctx)
	return p.Start(ctx)
}

// Kill kills immediately.
func (p *Process) Kill() error {
	p.mu.Lock()
	pid := p.pid
	p.mu.Unlock()
	if pid > 0 {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}
	return nil
}

// Terminate alias.
func (p *Process) Terminate() error { return p.Kill() }

func (p *Process) runLoop(ctx context.Context) {
	defer func() {
		p.mu.Lock()
		if p.done != nil {
			select {
			case <-p.done:
			default:
				close(p.done)
			}
		}
		p.mu.Unlock()
	}()
	defer closeStopped(p.stopped)

	pol := p.normalizedPolicy()
	if p.backoff == 0 {
		p.backoff = pol.InitialBackoff
		if p.backoff <= 0 {
			p.backoff = 100 * time.Millisecond
		}
	}

	first := true
	for {
		isRestart := !first

		// check restart limit before a restart attempt
		p.mu.Lock()
		now := time.Now()
		cutoff := now.Add(-pol.Window)
		filtered := p.restarts[:0]
		for _, t := range p.restarts {
			if t.After(cutoff) {
				filtered = append(filtered, t)
			}
		}
		p.restarts = filtered
		limitExceeded := false
		if isRestart && pol.Enabled && pol.MaxRestarts >= 0 && len(p.restarts) >= pol.MaxRestarts {
			limitExceeded = true
		}
		if isRestart && !limitExceeded {
			p.restarts = append(p.restarts, time.Now())
		}
		p.mu.Unlock()

		if limitExceeded {
			p.setState(StateFailed)
			if p.logger != nil {
				p.logger.Error("process exceeded restart limit",
					slog.String("process", p.Name()),
					slog.Int("restarts", len(p.restarts)),
					slog.Int("max", pol.MaxRestarts))
			}
			if p.supervisor != nil {
				p.supervisor.setDegraded(true)
			}
			return
		}

		// launch
		p.setState(StateStarting)
		err := p.runOnce(ctx)
		first = false
		p.mu.Lock()
		p.lastErr = err
		stopRequested := p.stopRequested
		p.mu.Unlock()

		if stopRequested || ctx.Err() != nil {
			p.setState(StateStopped)
			return
		}

		if err == nil {
			// process exited successfully
			if !pol.Enabled {
				p.setState(StateStopped)
				return
			}
		}

		if !pol.Enabled {
			if err != nil {
				p.setState(StateFailed)
			} else {
				p.setState(StateStopped)
			}
			return
		}

		p.setState(StateRestarting)
		wait := p.backoff
		// jitter
		if pol.Jitter > 0 {
			jitter := time.Duration(float64(wait) * pol.Jitter)
			if jitter > 0 {
				wait += time.Duration(rand.Int63n(int64(jitter) + 1)) //nolint:gosec // non-cryptographic jitter is sufficient
			}
		}
		if wait > pol.MaxBackoff && pol.MaxBackoff > 0 {
			wait = pol.MaxBackoff
		}
		if wait <= 0 {
			wait = 100 * time.Millisecond
		}

		select {
		case <-time.After(wait):
		case <-ctx.Done():
			p.setState(StateStopped)
			return
		case <-p.stopped:
			return
		}

		// increase backoff
		p.mu.Lock()
		if p.backoff <= 0 {
			p.backoff = pol.InitialBackoff
		}
		p.backoff = time.Duration(float64(p.backoff) * pol.Multiplier)
		if pol.MaxBackoff > 0 && p.backoff > pol.MaxBackoff {
			p.backoff = pol.MaxBackoff
		}
		p.mu.Unlock()
	}
}

func closeStopped(ch chan struct{}) {
	select {
	case <-ch:
	default:
		close(ch)
	}
}

func (p *Process) runOnce(ctx context.Context) error {
	spec := p.spec
	var cmdPath string
	if strings.ContainsRune(spec.Command, os.PathSeparator) {
		cmdPath = spec.Command
	} else {
		var err error
		cmdPath, err = exec.LookPath(spec.Command)
		if err != nil {
			p.lastErr = err
			return err
		}
	}
	// validate permissions/owner
	if fi, err := os.Stat(cmdPath); err == nil {
		if fi.Mode()&0o111 == 0 {
			err := fmt.Errorf("command not executable: %s", cmdPath)
			p.lastErr = err
			return err
		}
	}

	cmd := exec.CommandContext(ctx, cmdPath, p.args()...)
	cmd.Dir = spec.Dir

	env, secrets := buildEnv(spec, p.supervisor)
	cmd.Env = env

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGKILL,
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		p.lastErr = err
		return err
	}

	p.mu.Lock()
	p.cmd = cmd
	p.pid = cmd.Process.Pid
	p.pgid = cmd.Process.Pid
	p.exitCode = 0
	p.state = StateRunning
	p.mu.Unlock()

	if p.logger != nil {
		p.logger.Info("started process",
			slog.String("process", p.Name()),
			slog.Int("pid", p.pid),
			slog.String("command", redactCommand(cmdPath, secrets)),
		)
	}

	// capture output
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		p.captureOutput(stdout, "stdout", secrets, spec.Stdout)
	}()
	go func() {
		defer wg.Done()
		p.captureOutput(stderr, "stderr", secrets, spec.Stderr)
	}()

	waitErr := cmd.Wait()
	wg.Wait()

	p.mu.Lock()
	p.cmd = nil
	if waitErr != nil {
		if ee, ok := waitErr.(*exec.ExitError); ok {
			p.exitCode = ee.ExitCode()
			if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
				p.signal = ws.Signal()
			}
		}
	} else {
		p.exitCode = 0
	}
	p.lastErr = waitErr
	p.mu.Unlock()

	if p.logger != nil {
		if waitErr != nil {
			p.logger.Warn("process exited",
				slog.String("process", p.Name()),
				slog.Int("pid", p.pid),
				slog.Any("error", waitErr),
				slog.Int("exit_code", p.exitCode),
			)
		} else {
			p.logger.Info("process exited",
				slog.String("process", p.Name()),
				slog.Int("pid", p.pid),
				slog.Int("exit_code", p.exitCode),
			)
		}
	}
	return waitErr
}

func (p *Process) args() []string {
	spec := p.spec
	if len(spec.Args) > 0 {
		return spec.Args
	}
	if len(spec.Argv) > 0 {
		return spec.Argv
	}
	if len(spec.Arguments) > 0 {
		return spec.Arguments
	}
	return nil
}

func (p *Process) captureOutput(r io.ReadCloser, stream string, secrets []string, w io.Writer) {
	defer func() { _ = r.Close() }()
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	const maxBytes = 1 << 20
	total := 0
	for scanner.Scan() {
		line := scanner.Text()
		if total+len(line) > maxBytes {
			if p.logger != nil {
				p.logger.Warn("process output truncated",
					slog.String("process", p.Name()),
					slog.String("stream", stream))
			}
			break
		}
		total += len(line)
		redacted := redactString(line, secrets)
		if w != nil {
			_, _ = fmt.Fprintln(w, redacted)
		}
		if p.logger != nil {
			p.logger.Info("process output",
				slog.String("process", p.Name()),
				slog.String("stream", stream),
				slog.String("line", redacted))
		}
	}
}

func (p *Process) normalizedPolicy() RestartPolicy {
	pol := p.pol
	pol = mergePolicy(pol, p.spec.Restart)
	pol = mergePolicy(pol, p.spec.RestartPolicy)
	pol = mergePolicy(pol, p.spec.Policy)
	if p.spec.MaxRestarts != 0 {
		pol.MaxRestarts = p.spec.MaxRestarts
	}
	if p.spec.Window != 0 {
		pol.Window = p.spec.Window
	}
	if !pol.Enabled && (pol.MaxRestarts != 0 || pol.Window != 0 || pol.InitialBackoff != 0 || pol.MaxBackoff != 0) {
		pol.Enabled = true
	}
	if pol.InitialBackoff == 0 {
		pol.InitialBackoff = 100 * time.Millisecond
	}
	if pol.MaxBackoff == 0 {
		pol.MaxBackoff = 5 * time.Second
	}
	if pol.Multiplier == 0 {
		pol.Multiplier = 2
	}
	if pol.Window == 0 {
		pol.Window = time.Minute
	}
	if pol.Jitter == 0 && !pol.JitterEnabled {
		pol.Jitter = 0.2
	}
	if pol.JitterEnabled && pol.Jitter == 0 {
		pol.Jitter = 0.2
	}
	return pol
}

// Supervisor methods

func (s *Supervisor) setDegraded(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Supervisor itself has no degraded flag exposed; log only.
	if v && s.logger != nil {
		s.logger.Warn("supervisor degraded")
	}
}

func (s *Supervisor) track(p *Process) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.processes == nil {
		s.processes = make(map[string]*Process)
	}
	s.processes[p.Name()] = p
}

func (s *Supervisor) untrack(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.processes, name)
}

// Add registers a process without starting it.
func (s *Supervisor) Add(args ...any) (*Process, error) {
	spec := s.parseSpec(args...)
	p := s.newProcess(spec)
	s.track(p)
	return p, nil
}

// Aliases
func (s *Supervisor) AddProcess(args ...any) (*Process, error) { return s.Add(args...) }
func (s *Supervisor) Register(args ...any) (*Process, error)   { return s.Add(args...) }
func (s *Supervisor) RegisterProcess(args ...any) (*Process, error) {
	return s.Add(args...)
}
func (s *Supervisor) Create(args ...any) (*Process, error) { return s.Add(args...) }
func (s *Supervisor) CreateProcess(args ...any) (*Process, error) {
	return s.Add(args...)
}
func (s *Supervisor) NewProcess(args ...any) (*Process, error) { return s.Add(args...) }

// Start creates (if needed) and starts a process.
func (s *Supervisor) Start(ctx context.Context, args ...any) (*Process, error) {
	if len(args) == 0 {
		s.mu.Lock()
		list := make([]*Process, 0, len(s.processes))
		for _, p := range s.processes {
			list = append(list, p)
		}
		s.mu.Unlock()
		for _, p := range list {
			_ = p.Start(ctx)
		}
		return nil, nil
	}
	// If first arg is existing process, start it
	if len(args) > 0 {
		if p, ok := args[0].(*Process); ok && len(args) == 1 {
			if err := p.Start(ctx); err != nil {
				return p, err
			}
			s.track(p)
			return p, nil
		}
	}
	spec := s.parseSpec(args...)
	// reuse existing
	s.mu.Lock()
	if existing, ok := s.processes[spec.Name]; ok {
		s.mu.Unlock()
		if err := existing.Start(ctx); err != nil {
			return existing, err
		}
		return existing, nil
	}
	s.mu.Unlock()
	p := s.newProcess(spec)
	s.track(p)
	if err := p.Start(ctx); err != nil {
		s.untrack(p.Name())
		return p, err
	}
	return p, nil
}

// Aliases
func (s *Supervisor) Run(ctx context.Context, args ...any) (*Process, error) {
	return s.Start(ctx, args...)
}
func (s *Supervisor) Launch(ctx context.Context, args ...any) (*Process, error) {
	return s.Start(ctx, args...)
}
func (s *Supervisor) Spawn(ctx context.Context, args ...any) (*Process, error) {
	return s.Start(ctx, args...)
}
func (s *Supervisor) Supervise(ctx context.Context, args ...any) (*Process, error) {
	return s.Start(ctx, args...)
}
func (s *Supervisor) Manage(ctx context.Context, args ...any) (*Process, error) {
	return s.Start(ctx, args...)
}
func (s *Supervisor) StartProcess(ctx context.Context, args ...any) (*Process, error) {
	return s.Start(ctx, args...)
}
func (s *Supervisor) RunProcess(ctx context.Context, args ...any) (*Process, error) {
	return s.Start(ctx, args...)
}
func (s *Supervisor) LaunchProcess(ctx context.Context, args ...any) (*Process, error) {
	return s.Start(ctx, args...)
}
func (s *Supervisor) SpawnProcess(ctx context.Context, args ...any) (*Process, error) {
	return s.Start(ctx, args...)
}

// Stop stops a process (or all).
func (s *Supervisor) Stop(ctx context.Context, args ...any) error {
	if len(args) == 0 {
		return s.StopAll(ctx)
	}
	for _, a := range args {
		switch v := a.(type) {
		case *Process:
			_ = v.Stop(ctx)
		case string:
			s.mu.Lock()
			p := s.processes[v]
			s.mu.Unlock()
			if p != nil {
				_ = p.Stop(ctx)
			}
		}
	}
	return nil
}

// Aliases
func (s *Supervisor) StopProcess(ctx context.Context, args ...any) error {
	return s.Stop(ctx, args...)
}
func (s *Supervisor) Kill(ctx context.Context, args ...any) error {
	if len(args) == 0 {
		return s.StopAll(ctx)
	}
	for _, a := range args {
		if p, ok := a.(*Process); ok { //nolint:gocritic
			_ = p.Kill()
		} else if name, ok := a.(string); ok {
			s.mu.Lock()
			p := s.processes[name]
			s.mu.Unlock()
			if p != nil {
				_ = p.Kill()
			}
		}
	}
	return nil
}
func (s *Supervisor) Terminate(ctx context.Context, args ...any) error {
	return s.Kill(ctx, args...)
}
func (s *Supervisor) Shutdown(ctx context.Context) error { return s.StopAll(ctx) }
func (s *Supervisor) StopAll(ctx context.Context) error {
	s.mu.Lock()
	list := make([]*Process, 0, len(s.processes))
	for _, p := range s.processes {
		list = append(list, p)
	}
	s.mu.Unlock()
	for _, p := range list {
		_ = p.Stop(ctx)
	}
	return nil
}

// Get returns a process by name.
func (s *Supervisor) Get(name string) (*Process, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.processes[name]
	return p, ok
}

// Aliases
func (s *Supervisor) Lookup(name string) (*Process, bool)  { return s.Get(name) }
func (s *Supervisor) Find(name string) (*Process, bool)    { return s.Get(name) }
func (s *Supervisor) Process(name string) (*Process, bool) { return s.Get(name) }
func (s *Supervisor) GetProcess(name string) (*Process, bool) {
	return s.Get(name)
}

// List returns all processes.
func (s *Supervisor) List() []*Process {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Process, 0, len(s.processes))
	for _, p := range s.processes {
		out = append(out, p)
	}
	return out
}

// Aliases
func (s *Supervisor) Processes() []*Process { return s.List() }
func (s *Supervisor) All() []*Process       { return s.List() }
func (s *Supervisor) AllProcesses() []*Process {
	return s.List()
}

// Status returns map of statuses.
func (s *Supervisor) Status() map[string]ProcessStatus {
	out := make(map[string]ProcessStatus)
	for _, p := range s.List() {
		out[p.Name()] = p.Status()
	}
	return out
}

// Service interface implementation.
func (s *Supervisor) Name() string { return "supervisor" }
func (s *Supervisor) StartService(ctx context.Context) error {
	return nil
}
func (s *Supervisor) StartSvc(ctx context.Context) error { return s.StartService(ctx) }
func (s *Supervisor) StopService(ctx context.Context) error {
	return s.StopAll(ctx)
}
func (s *Supervisor) Health(ctx context.Context) contracts.ServiceInfo {
	return contracts.ServiceInfo{Name: "supervisor", State: contracts.ServiceRunning, Health: contracts.HealthOK}
}

// Process Service interface
func (p *Process) Health(ctx context.Context) contracts.ServiceInfo {
	st := p.Status()
	health := contracts.HealthOK
	state := contracts.ServiceRunning
	if st.State == string(StateFailed) {
		health = contracts.HealthFail
		state = contracts.ServiceFailed
	} else if st.State == string(StateDegraded) {
		health = contracts.HealthWarn
		state = contracts.ServiceDegraded
	}
	return contracts.ServiceInfo{Name: p.Name(), State: state, Health: health}
}

// Helpers: env

//nolint:gocritic // value semantics
func buildEnv(spec ProcessSpec, sup *Supervisor) ([]string, []string) {
	allow := spec.EnvAllowlist
	if len(allow) == 0 && sup != nil {
		allow = sup.envAllow
	}
	if len(allow) == 0 {
		allow = []string{"PATH", "HOME", "USER", "LOGNAME", "SHELL", "LANG", "LC_*", "TERM", "DISPLAY", "WAYLAND_DISPLAY", "SWAYSOCK", "XDG_*", "XAUTHORITY", "TMPDIR", "TZ"}
	}

	// filter os.Environ
	result := make(map[string]string)
	for _, e := range os.Environ() {
		k, v, ok := splitEnv(e)
		if !ok {
			continue
		}
		if nameAllowed(k, allow) {
			result[k] = v
		}
	}
	// spec.Env overrides
	if spec.Env != nil {
		for k, v := range spec.Env {
			if nameAllowed(k, allow) || true {
				result[k] = v
			}
		}
	}
	// EnvList
	for _, e := range spec.EnvList {
		k, v, ok := splitEnv(e)
		if !ok {
			continue
		}
		if nameAllowed(k, allow) || true {
			result[k] = v
		}
	}

	// collect secrets
	var secrets []string
	for k, v := range result {
		if isSensitive(k) && v != "" {
			secrets = append(secrets, v)
		}
	}

	out := make([]string, 0, len(result))
	for k, v := range result {
		out = append(out, k+"="+v)
	}
	return out, secrets
}

func splitEnv(e string) (name, value string, ok bool) {
	i := strings.IndexByte(e, '=')
	if i < 0 {
		return "", "", false
	}
	return e[:i], e[i+1:], true
}

func nameAllowed(name string, allow []string) bool {
	for _, pat := range allow {
		if pat == "*" {
			return true
		}
		if pat == name {
			return true
		}
		if ok, _ := filepath.Match(pat, name); ok {
			return true
		}
	}
	return false
}

func isSensitive(key string) bool {
	u := strings.ToUpper(key)
	return strings.Contains(u, "TOKEN") || strings.Contains(u, "SECRET") || strings.Contains(u, "PASSWORD")
}

var sensitiveKVRe = regexp.MustCompile(`(?i)([A-Za-z0-9_]*(?:TOKEN|SECRET|PASSWORD)[A-Za-z0-9_]*\s*=\s*)([^\s]+)`)

func redactString(s string, secrets []string) string {
	for _, v := range secrets {
		if v != "" && len(v) >= 3 {
			s = strings.ReplaceAll(s, v, "[REDACTED]")
		}
	}
	s = sensitiveKVRe.ReplaceAllString(s, "${1}[REDACTED]")
	return s
}

func redactCommand(cmd string, secrets []string) string {
	return redactString(cmd, secrets)
}

// Package-level helpers.
func RunProcess(ctx context.Context, args ...any) (*Process, error) {
	s := New(nil)
	return s.Start(ctx, args...)
}
func StartProcess(ctx context.Context, args ...any) (*Process, error) {
	return RunProcess(ctx, args...)
}
func Supervise(ctx context.Context, args ...any) (*Process, error) {
	return RunProcess(ctx, args...)
}
