package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Current schema versions for managed config files.
const (
	CurrentDesktopVersion   = 1
	CurrentPluginsVersion   = 1
	CurrentServicesVersion  = 1
	CurrentShortcutsVersion = 1
)

// LoadIssue describes a non-fatal config problem (invalid source, migration, etc.).
type LoadIssue struct {
	File   string
	Layer  string
	Err    error
	Action string
}

func (i LoadIssue) Error() string {
	if i.Err == nil && i.Action == "" {
		return ""
	}
	msg := ""
	if i.Err != nil {
		msg = i.Err.Error()
	}
	if i.File != "" && msg != "" && !strings.Contains(msg, i.File+":") && !strings.HasPrefix(msg, filepath.Base(i.File)+":") {
		// formatConfigError may already include file context.
		if !strings.HasPrefix(msg, "line ") {
			msg = i.File + ": " + msg
		}
	}
	if i.Action != "" {
		if msg != "" {
			msg += "; " + i.Action
		} else {
			msg = i.Action
		}
	}
	return msg
}

// LoadReport is the result of loading configuration with fallback.
type LoadReport struct {
	Desktop  DesktopConfig
	Degraded bool
	Issues   []LoadIssue
	// Source describes which config was used: "merged", "last-valid", or "defaults".
	Source string
}

// ActionableError returns a single actionable error string, or empty if healthy.
func (r LoadReport) ActionableError() string {
	if len(r.Issues) == 0 {
		return ""
	}
	parts := make([]string, 0, len(r.Issues))
	for _, is := range r.Issues {
		s := is.Error()
		if s != "" {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " | ")
}

// LoadDesktopWithFallback loads desktop.yaml across layers. If any file layer
// or the merged result is invalid/unparseable, it falls back to the last valid
// cached config (if present) or embedded defaults, marks the report degraded,
// and returns an actionable error. It never returns an error for invalid user
// config (only for unrecoverable internal failures).
func LoadDesktopWithFallback(opts LoadOptions) (LoadReport, error) {
	if opts.Layers == nil {
		opts.Layers = DefaultLayers()
	}
	var issues []LoadIssue

	// Attempt layered load with per-file validation/migration.
	merged := append([]byte(nil), DefaultDesktop...)
	for _, l := range opts.Layers {
		if l.Path == "" {
			continue
		}
		p := filepath.Join(l.Path, "desktop.yaml")
		b, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			issues = append(issues, LoadIssue{File: p, Layer: l.Name, Err: fmt.Errorf("read config: %w", err), Action: "check file permissions"})
			return fallbackDesktop(issues)
		}
		migrated, _, migIssues := migrateIfNeeded("desktop.yaml", p, b, CurrentDesktopVersion)
		issues = append(issues, migIssues...)
		// Even if migration "changed" logically in registry (future), we use
		// migrated bytes; on failure migrateIfNeeded returns original+issues.
		if len(migIssues) == 0 || !hasFatalMigrationIssue(migIssues) {
			b = migrated
		}
		if _, err := ParseDesktop(b); err != nil {
			issues = append(issues, LoadIssue{File: p, Layer: l.Name, Err: formatConfigError(p, err), Action: "fix the config file or remove it to use defaults"})
			return fallbackDesktop(issues)
		}
		merged, err = MergeYAML(merged, b)
		if err != nil {
			issues = append(issues, LoadIssue{File: p, Layer: l.Name, Err: fmt.Errorf("merge config: %w", err), Action: "fix YAML structure"})
			return fallbackDesktop(issues)
		}
	}

	// Apply env/flags overlays to the merged base.
	if opts.Env {
		envOverlay, err := EnvOverlay("desktop.yaml")
		if err != nil {
			issues = append(issues, LoadIssue{Layer: "env", Err: err, Action: "check I3NEXT_DESKTOP_* environment variables"})
			return fallbackDesktop(issues)
		}
		if len(envOverlay) > 0 {
			merged, err = MergeYAML(merged, envOverlay)
			if err != nil {
				issues = append(issues, LoadIssue{Layer: "env", Err: err})
				return fallbackDesktop(issues)
			}
		}
	}
	if len(opts.Flags) > 0 {
		flagOverlay, err := FlagsOverlay("desktop.yaml", opts.Flags)
		if err != nil {
			issues = append(issues, LoadIssue{Layer: "flags", Err: err, Action: "check command-line config overrides"})
			return fallbackDesktop(issues)
		}
		if len(flagOverlay) > 0 {
			merged, err = MergeYAML(merged, flagOverlay)
			if err != nil {
				issues = append(issues, LoadIssue{Layer: "flags", Err: err})
				return fallbackDesktop(issues)
			}
		}
	}

	cfg, err := ParseDesktop(merged)
	if err != nil {
		issues = append(issues, LoadIssue{File: "desktop.yaml", Layer: "merged", Err: formatConfigError("desktop.yaml", err), Action: "check env/flag overrides and config files"})
		return fallbackDesktop(issues)
	}

	// Persist last valid merged config (best effort).
	_ = saveLastValid("desktop.yaml", merged)

	return LoadReport{Desktop: cfg, Degraded: len(issues) > 0, Issues: issues, Source: "merged"}, nil
}

func hasFatalMigrationIssue(issues []LoadIssue) bool {
	for _, is := range issues {
		if is.Err == nil {
			continue
		}
		// Informational "migrated..." issues are not fatal; load may proceed
		// with migrated bytes (when migration succeeded and wrote file) or
		// original handling decided by caller. migrateIfNeeded returns fatal
		// issues only when it aborts and returns original.
		msg := is.Err.Error()
		if strings.Contains(msg, "migrate v") || strings.Contains(msg, "backup before migration") || strings.Contains(msg, "write migrated config") || strings.Contains(msg, "unsupported config version") {
			return true
		}
	}
	return false
}

func fallbackDesktop(issues []LoadIssue) (LoadReport, error) {
	// Try last valid cache.
	if b, err := loadLastValid("desktop.yaml"); err == nil && len(b) > 0 {
		if cfg, err := ParseDesktop(b); err == nil {
			issues = append(issues, LoadIssue{Err: errors.New("using last valid cached config"), Action: "resolve the config error above to use updated settings"})
			return LoadReport{Desktop: cfg, Degraded: true, Issues: issues, Source: "last-valid"}, nil
		}
	}
	// Fall back to embedded defaults.
	cfg, err := ParseDesktop(DefaultDesktop)
	if err != nil {
		return LoadReport{}, fmt.Errorf("internal: embedded default desktop config invalid: %w", err)
	}
	issues = append(issues, LoadIssue{Err: errors.New("using embedded default config"), Action: "resolve the config error above"})
	return LoadReport{Desktop: cfg, Degraded: true, Issues: issues, Source: "defaults"}, nil
}

func lastValidDir() string {
	return filepath.Join(StateDir(), "config", "last-valid")
}

func lastValidPath(name string) string {
	return filepath.Join(lastValidDir(), name)
}

func saveLastValid(name string, data []byte) error {
	dir := lastValidDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return err
	}
	return AtomicWrite(lastValidPath(name), data, 0o600)
}

func loadLastValid(name string) ([]byte, error) {
	return os.ReadFile(lastValidPath(name))
}

// formatConfigError produces an actionable message including file and, when
// available, line/key information from yaml errors.
func formatConfigError(file string, err error) error {
	if err == nil {
		return nil
	}
	base := filepath.Base(file)
	msg := err.Error()
	// Strip common yaml.v3 prefixes and wrapper text, keep line info.
	msg = strings.ReplaceAll(msg, "yaml: unmarshal errors:\n  ", "")
	msg = strings.ReplaceAll(msg, "\n  ", "; ")
	msg = strings.TrimPrefix(msg, "yaml: ")
	msg = strings.TrimPrefix(msg, "desktop config: ")
	msg = strings.TrimPrefix(msg, "plugins config: ")
	msg = strings.TrimPrefix(msg, "services config: ")
	msg = strings.TrimPrefix(msg, "shortcuts config: ")
	msg = strings.TrimPrefix(msg, "unmarshal errors:; ")
	msg = strings.TrimPrefix(msg, "unmarshal errors: ")
	// If the message already contains the base file, keep it; otherwise prefix.
	if base != "" && !strings.Contains(msg, base+":") && !strings.Contains(msg, base+" ") {
		if strings.HasPrefix(msg, "line ") {
			msg = base + ":" + strings.TrimPrefix(msg, "line ")
		} else {
			msg = base + ": " + msg
		}
	}
	return errors.New(msg)
}

// migrateIfNeeded applies versioned migrations to a config file's bytes.
// It creates an automatic backup before writing migrated content back to path
// when migration changes the document. Returns the (possibly migrated) bytes,
// whether content changed, and any issues.
func migrateIfNeeded(name, path string, data []byte, current int) ([]byte, bool, []LoadIssue) {
	v, err := readVersion(data)
	if err != nil {
		return data, false, nil
	}
	if v >= current {
		return data, false, nil
	}
	if v <= 0 {
		v = 1
	}
	orig := append([]byte(nil), data...)
	out := data
	changed := false
	var issues []LoadIssue
	for v < current {
		next, ok, merr := migrateOne(name, v, out)
		if merr != nil {
			issues = append(issues, LoadIssue{File: path, Err: fmt.Errorf("migrate v%d->v%d: %w", v, v+1, merr), Action: "update the config file manually"})
			return orig, false, issues
		}
		if !ok {
			issues = append(issues, LoadIssue{File: path, Err: fmt.Errorf("unsupported config version %d (current %d)", v, current), Action: "update the config file to a supported version"})
			return orig, false, issues
		}
		out = next
		changed = true
		v++
	}
	if !changed {
		return orig, false, issues
	}
	if err := backupFile(path, orig); err != nil {
		issues = append(issues, LoadIssue{File: path, Err: fmt.Errorf("backup before migration: %w", err), Action: "check directory permissions"})
		return orig, false, issues
	}
	if err := AtomicWrite(path, out, 0o644); err != nil {
		issues = append(issues, LoadIssue{File: path, Err: fmt.Errorf("write migrated config: %w", err), Action: "check directory permissions"})
		return orig, false, issues
	}
	issues = append(issues, LoadIssue{File: path, Err: fmt.Errorf("migrated config from v%d to v%d (backup created)", readVersionOr(orig, 0), current)})
	return out, true, issues
}

func readVersion(data []byte) (int, error) {
	var doc struct {
		Version int `yaml:"version"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return 0, err
	}
	return doc.Version, nil
}

func readVersionOr(data []byte, def int) int {
	v, err := readVersion(data)
	if err != nil || v == 0 {
		return def
	}
	return v
}

// migrationFunc transforms a config document from version `from` to `from+1`.
type migrationFunc func(data []byte) ([]byte, error)

var migrationRegistry = map[string]map[int]migrationFunc{
	// Future migrations register here, e.g.:
	// "desktop.yaml": {1: migrateDesktopV1ToV2},
}

func migrateOne(name string, from int, data []byte) ([]byte, bool, error) {
	byName := migrationRegistry[name]
	if byName == nil {
		return nil, false, nil
	}
	fn := byName[from]
	if fn == nil {
		return nil, false, nil
	}
	out, err := fn(data)
	if err != nil {
		return nil, true, err
	}
	return out, true, nil
}

// MigrateBytes migrates a config document of the given name from its version
// to current, returning migrated bytes and whether a change occurred. It does
// not write files or create backups.
func MigrateBytes(name string, data []byte) ([]byte, bool, error) {
	current, ok := currentVersion(name)
	if !ok {
		return nil, false, fmt.Errorf("unknown config file %q", name)
	}
	v, err := readVersion(data)
	if err != nil {
		return nil, false, err
	}
	if v >= current {
		return append([]byte(nil), data...), false, nil
	}
	if v <= 0 {
		v = 1
	}
	out := append([]byte(nil), data...)
	changed := false
	for v < current {
		next, ok, merr := migrateOne(name, v, out)
		if merr != nil {
			return nil, false, merr
		}
		if !ok {
			return nil, false, fmt.Errorf("no migration path for %s v%d->v%d", name, v, current)
		}
		out = next
		changed = true
		v++
	}
	return out, changed, nil
}

func currentVersion(name string) (int, bool) {
	switch name {
	case "desktop.yaml":
		return CurrentDesktopVersion, true
	case "plugins.yaml":
		return CurrentPluginsVersion, true
	case "services.yaml":
		return CurrentServicesVersion, true
	case "shortcuts.yaml":
		return CurrentShortcutsVersion, true
	default:
		return 0, false
	}
}

func backupFile(path string, data []byte) error {
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	ts := time.Now().UTC().Format("20060102T150405Z")
	bak := filepath.Join(dir, base+".bak."+ts)
	if _, err := os.Stat(bak); err == nil {
		bak = filepath.Join(dir, fmt.Sprintf("%s.bak.%s.%d", base, ts, os.Getpid()))
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	return AtomicWrite(bak, data, 0o644)
}

// ValidateDesktopFile reads and strictly validates a desktop config file.
func ValidateDesktopFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: read: %w", path, err)
	}
	if _, err := ParseDesktop(b); err != nil {
		return formatConfigError(path, err)
	}
	return nil
}

// ValidateFile reads and strictly validates a named config file.
func ValidateFile(name, path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: read: %w", path, err)
	}
	var perr error
	switch name {
	case "desktop.yaml":
		_, perr = ParseDesktop(b)
	case "plugins.yaml":
		_, perr = ParsePlugins(b)
	case "services.yaml":
		_, perr = ParseServices(b)
	case "shortcuts.yaml":
		_, perr = ParseShortcuts(b)
	default:
		return fmt.Errorf("unknown config file %q", name)
	}
	if perr != nil {
		return formatConfigError(path, perr)
	}
	return nil
}

// ensure bytes import used
var _ = bytes.TrimSpace
