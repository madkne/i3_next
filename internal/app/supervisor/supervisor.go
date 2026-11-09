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

var (
	ErrNotFound       = errors.New("process not found")
	ErrInvalidArg     = errors.New("invalid argument")
	ErrAlreadyRunning = errors.New("process already running")
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
	Enabled        bool
	MaxRestarts    int
	Window         time.Duration
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Multiplier     float64
	Jitter         float64
}

// ProcessSpec describes a child process.
type ProcessSpec struct {
	Name         string
	ID           string
	Command      string
	Args         []string
	Env          map[string]string
	EnvAllowlist []string
	Dir          string
	GracePeriod  time.Duration
	Restart      RestartPolicy

	Logger     *slog.Logger
	Stdout     io.Writer
	Stderr     io.Writer
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

func WithEnvAllowlist(list []string) Option {
	return func(s *Supervisor) {
		if len(list) > 0 {
			s.envAllow = append([]string(nil), list...)
		}
	}
}

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

//nolint:gocritic // value semantics
func (s *Supervisor) newProcess(spec ProcessSpec) *Process {
	spec = s.parseSpec(spec)
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
	return p
}

//nolint:gocritic // value semantics
func (s *Supervisor) parseSpec(spec ProcessSpec) ProcessSpec {
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
	if spec.GracePeriod == 0 {
		spec.GracePeriod = s.gracePeriod
	}
	if spec.GracePeriod == 0 {
		spec.GracePeriod = 5 * time.Second
	}
	if spec.Logger == nil {
		spec.Logger = s.logger
	}
	if spec.Supervisor == nil {
		spec.Supervisor = s
	}
	return spec
}

//nolint:gocritic // value semantics
func mergePolicy(a, b RestartPolicy) RestartPolicy {
	if b.Enabled {
		a.Enabled = true
	}
	if b.MaxRestarts != 0 {
		a.MaxRestarts = b.MaxRestarts
	}
	if b.Window != 0 {
		a.Window = b.Window
	}
	if b.InitialBackoff != 0 {
		a.InitialBackoff = b.InitialBackoff
	}
	if b.MaxBackoff != 0 {
		a.MaxBackoff = b.MaxBackoff
	}
	if b.Multiplier != 0 {
		a.Multiplier = b.Multiplier
	}
	if b.Jitter != 0 {
		a.Jitter = b.Jitter
	}
	return a
}

// Add registers a process without starting it.
//
//nolint:gocritic // value semantics
func (s *Supervisor) Add(spec ProcessSpec) (*Process, error) {
	p := s.newProcess(spec)
	s.track(p)
	return p, nil
}

// Start creates (if needed) and starts a process.
//
//nolint:gocritic // value semantics
func (s *Supervisor) Start(ctx context.Context, spec ProcessSpec) (*Process, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// Normalise first: track() keys processes by the normalised name, so the
	// lookup below must use the same key or a second Start spawns a duplicate.
	spec = s.parseSpec(spec)
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

// Stop stops the named process.
func (s *Supervisor) Stop(ctx context.Context, name string) error {
	s.mu.Lock()
	p := s.processes[name]
	s.mu.Unlock()
	if p == nil {
		return fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	return p.Stop(ctx)
}

// StopAll stops every tracked process.
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
func (s *Supervisor) StopService(ctx context.Context) error {
	return s.StopAll(ctx)
}
func (s *Supervisor) Health(ctx context.Context) contracts.ServiceInfo {
	return contracts.ServiceInfo{Name: "supervisor", State: contracts.ServiceRunning, Health: contracts.HealthOK}
}

func (s *Supervisor) setDegraded(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
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

// Start begins supervision.
func (p *Process) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	p.mu.Lock()
	if p.cancel != nil {
		p.mu.Unlock()
		return ErrAlreadyRunning
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
	p.stopRequested = true
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

// Health returns the process health as a service info.
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
	if pol.Jitter == 0 {
		pol.Jitter = 0.2
	}
	return pol
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
	if spec.Env != nil {
		for k, v := range spec.Env {
			result[k] = v
		}
	}

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
