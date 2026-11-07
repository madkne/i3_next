package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withIsolatedXDG(t *testing.T) (configHome, stateHome string) {
	t.Helper()
	root := t.TempDir()
	configHome = filepath.Join(root, "config")
	stateHome = filepath.Join(root, "state")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_STATE_HOME", stateHome)
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(root, "run"))
	return configHome, stateHome
}

func validDesktopYAML(level string) []byte {
	return []byte("version: 1\nsession:\n  name: i3_next\n  compositor: sway\npaths:\n  config_dir: ~/.config/i3-next\n  data_dir: ~/.local/share/i3-next\n  state_dir: ~/.local/state/i3-next\n  cache_dir: ~/.cache/i3-next\nlogging:\n  level: " + level + "\n  format: text\nui:\n  shell: quickshell\n  theme: default\n")
}

func TestLoadDesktopWithFallbackInvalidConfigUsesDefaults(t *testing.T) {
	configHome, _ := withIsolatedXDG(t)
	userDir := filepath.Join(configHome, "i3-next")
	if err := os.MkdirAll(userDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// Invalid: unknown field (strict), actionable message expected.
	if err := os.WriteFile(filepath.Join(userDir, "desktop.yaml"), []byte("version: 1\nnope: true\n"), 0o644); err != nil { //nolint:gosec // test file permissions
		t.Fatal(err)
	}
	report, err := LoadDesktopWithFallback(LoadOptions{})
	if err != nil {
		t.Fatalf("unexpected hard error for invalid config: %v", err)
	}
	if !report.Degraded {
		t.Fatal("expected degraded report")
	}
	if report.Source != "defaults" {
		t.Fatalf("source=%q, want defaults (no last-valid yet)", report.Source)
	}
	if report.Desktop.Session.Compositor != "sway" {
		t.Fatalf("expected default config loaded, got %+v", report.Desktop)
	}
	msg := report.ActionableError()
	if msg == "" {
		t.Fatal("expected actionable error")
	}
	if !strings.Contains(msg, "desktop.yaml") {
		t.Fatalf("actionable error missing file: %q", msg)
	}
	// yaml.v3 reports line info for unknown fields in nested/strict decode context; at minimum include field/path hint.
	if !strings.Contains(msg, "field") && !strings.Contains(msg, "line") && !strings.Contains(msg, "nope") {
		t.Fatalf("actionable error missing field/line context: %q", msg)
	}
}

func TestLoadDesktopWithFallbackInvalidUsesLastValid(t *testing.T) {
	configHome, _ := withIsolatedXDG(t)
	userDir := filepath.Join(configHome, "i3-next")
	if err := os.MkdirAll(userDir, 0o700); err != nil {
		t.Fatal(err)
	}
	// First: valid config becomes last-valid.
	if err := os.WriteFile(filepath.Join(userDir, "desktop.yaml"), validDesktopYAML("warn"), 0o644); err != nil { //nolint:gosec // test file permissions
		t.Fatal(err)
	}
	okReport, err := LoadDesktopWithFallback(LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if okReport.Degraded || okReport.Source != "merged" {
		t.Fatalf("expected healthy merged load, got degraded=%v source=%q issues=%v", okReport.Degraded, okReport.Source, okReport.Issues)
	}
	if okReport.Desktop.Logging.Level != "warn" {
		t.Fatalf("level=%q, want warn", okReport.Desktop.Logging.Level)
	}
	// Corrupt config: should fall back to last valid (warn), not defaults.
	if err := os.WriteFile(filepath.Join(userDir, "desktop.yaml"), []byte("::not yaml:: ["), 0o644); err != nil { //nolint:gosec // test file permissions
		t.Fatal(err)
	}
	bad, err := LoadDesktopWithFallback(LoadOptions{})
	if err != nil {
		t.Fatalf("unexpected hard error: %v", err)
	}
	if !bad.Degraded {
		t.Fatal("expected degraded")
	}
	if bad.Source != "last-valid" {
		t.Fatalf("source=%q, want last-valid", bad.Source)
	}
	if bad.Desktop.Logging.Level != "warn" {
		t.Fatalf("expected last-valid config (warn), got %q", bad.Desktop.Logging.Level)
	}
	if bad.ActionableError() == "" {
		t.Fatal("expected actionable error")
	}
}

func TestLoadDesktopWithFallbackCachesLastValidWith0600(t *testing.T) {
	configHome, stateHome := withIsolatedXDG(t)
	userDir := filepath.Join(configHome, "i3-next")
	if err := os.MkdirAll(userDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "desktop.yaml"), validDesktopYAML("info"), 0o644); err != nil { //nolint:gosec // test file permissions
		t.Fatal(err)
	}
	_, err := LoadDesktopWithFallback(LoadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(stateHome, "i3-next", "config", "last-valid", "desktop.yaml")
	info, err := os.Stat(p)
	if err != nil {
		t.Fatalf("last-valid not written: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("last-valid mode=%o, want 600", info.Mode().Perm())
	}
}

func TestValidateFileActionable(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "desktop.yaml")
	if err := os.WriteFile(p, []byte("version: 1\nlogging:\n  level: info\n  unknown: x\n"), 0o644); err != nil { //nolint:gosec // test file permissions
		t.Fatal(err)
	}
	err := ValidateFile("desktop.yaml", p)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "desktop.yaml") {
		t.Fatalf("error missing file: %v", err)
	}
}
