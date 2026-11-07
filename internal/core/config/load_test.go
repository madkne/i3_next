package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeYAMLMappingRecursive(t *testing.T) {
	base := []byte("logging:\n  level: info\n  format: text\nui:\n  theme: default\n")
	overlay := []byte("logging:\n  level: debug\n")
	out, err := MergeYAML(base, overlay)
	if err != nil {
		t.Fatal(err)
	}
	c, err := ParseDesktop(out)
	if err != nil {
		t.Fatalf("parse merged: %v\n%s", err, out)
	}
	if c.Logging.Level != "debug" {
		t.Fatalf("level = %q, want debug", c.Logging.Level)
	}
	if c.Logging.Format != "text" {
		t.Fatalf("format = %q, want text", c.Logging.Format)
	}
	if c.UI.Theme != "default" {
		t.Fatalf("theme = %q, want default", c.UI.Theme)
	}
}

func TestLoadDesktopMergedLayerOrder(t *testing.T) {
	root := t.TempDir()
	system := filepath.Join(root, "etc", "i3-next")
	user := filepath.Join(root, "home", ".config", "i3-next")
	if err := os.MkdirAll(system, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(user, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(system, "desktop.yaml"), []byte("logging:\n  level: warn\nui:\n  theme: system\n"), 0o644); err != nil { //nolint:gosec // test file permissions
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(user, "desktop.yaml"), []byte("logging:\n  level: error\n"), 0o644); err != nil { //nolint:gosec // test file permissions
		t.Fatal(err)
	}

	t.Setenv("I3NEXT_DESKTOP_LOGGING_LEVEL", "debug")
	opts := LoadOptions{
		Layers: []Layer{{Name: "system", Path: system}, {Name: "user", Path: user}},
		Env:    true,
		Flags:  map[string]string{"logging.level": "trace"},
	}
	c, _, err := LoadDesktopMerged(opts)
	if err != nil {
		t.Fatal(err)
	}
	if c.Logging.Level != "trace" {
		t.Fatalf("logging.level = %q, want trace (defaults < system < user < env < flags)", c.Logging.Level)
	}
	if c.UI.Theme != "system" {
		t.Fatalf("ui.theme = %q, want system (from system layer, preserved through merge)", c.UI.Theme)
	}
	if c.Session.Compositor != "sway" {
		t.Fatalf("session.compositor = %q, want sway (from defaults)", c.Session.Compositor)
	}
}

func TestLoadDesktopMergedPrecedenceWithoutFlags(t *testing.T) {
	root := t.TempDir()
	system := filepath.Join(root, "etc")
	user := filepath.Join(root, "user")
	if err := os.MkdirAll(system, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(user, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(system, "desktop.yaml"), []byte("logging:\n  level: warn\n"), 0o644); err != nil { //nolint:gosec // test file permissions
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(user, "desktop.yaml"), []byte("logging:\n  level: error\n"), 0o644); err != nil { //nolint:gosec // test file permissions
		t.Fatal(err)
	}
	t.Setenv("I3NEXT_DESKTOP_LOGGING_LEVEL", "debug")

	c, _, err := LoadDesktopMerged(LoadOptions{
		Layers: []Layer{{Name: "system", Path: system}, {Name: "user", Path: user}},
		Env:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if c.Logging.Level != "debug" {
		t.Fatalf("logging.level = %q, want debug (env overrides user)", c.Logging.Level)
	}
}

func TestInstallDefaultsOEXCL(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "config")
	if err := InstallDefaults(dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range FileNames {
		p := filepath.Join(dir, name)
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat %s: %v", name, err)
		}
		if info.Mode().Perm() != 0o644 {
			t.Fatalf("%s mode = %o, want 644", name, info.Mode().Perm())
		}
	}
	// Modify a file; second install must not overwrite.
	sentinel := []byte("version: 1\nsession:\n  name: custom\n  compositor: sway\npaths:\n  config_dir: x\n  data_dir: x\n  state_dir: x\n  cache_dir: x\nlogging:\n  level: info\n  format: text\nui:\n  shell: quickshell\n  theme: default\n")
	target := filepath.Join(dir, "desktop.yaml")
	if err := os.WriteFile(target, sentinel, 0o644); err != nil { //nolint:gosec // test file permissions
		t.Fatal(err)
	}
	if err := InstallDefaults(dir); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, sentinel) {
		t.Fatalf("desktop.yaml overwritten on second install")
	}
}

func TestInstallDefaultsIdempotent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "config")
	for i := 0; i < 3; i++ {
		if err := InstallDefaults(dir); err != nil {
			t.Fatalf("install %d: %v", i, err)
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != len(FileNames) {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("entries = %v, want %v", names, FileNames)
	}
}

func TestInstallDefaultsCreatesDir0700(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "config")
	if err := InstallDefaults(dir); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Fatalf("dir mode = %o, want 700", info.Mode().Perm())
	}
}

func TestEnvOverlay(t *testing.T) {
	t.Setenv("I3NEXT_DESKTOP_LOGGING_LEVEL", "warn")
	t.Setenv("I3NEXT_DESKTOP_UI_THEME", "dark")
	b, err := EnvOverlay("desktop.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "warn") {
		t.Fatalf("overlay missing warn: %s", b)
	}
}

func TestFlagsOverlay(t *testing.T) {
	b, err := FlagsOverlay("desktop.yaml", map[string]string{
		"logging.level":    "debug",
		"desktop.ui.theme": "solarized",
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if !strings.Contains(s, "debug") {
		t.Fatalf("overlay missing debug: %s", s)
	}
	if !strings.Contains(s, "solarized") {
		t.Fatalf("overlay missing solarized: %s", s)
	}
}

func TestDefaultLayersDocumented(t *testing.T) {
	layers := DefaultLayers()
	if len(layers) != 2 {
		t.Fatalf("len = %d, want 2 (system, user)", len(layers))
	}
	if layers[0].Name != "system" || layers[0].Path != "/etc/i3-next" {
		t.Fatalf("layer 0 = %+v, want system /etc/i3-next", layers[0])
	}
	if layers[1].Name != "user" || layers[1].Path != ConfigDir() {
		t.Fatalf("layer 1 = %+v, want user %s", layers[1], ConfigDir())
	}
}
