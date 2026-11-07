package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestMigrateBytesGolden(t *testing.T) {
	// Register a temporary v1->v2 migration for a synthetic desktop-like doc.
	// Keep registry mutation isolated to this test.
	orig := migrationRegistry["desktop.yaml"]
	t.Cleanup(func() { migrationRegistry["desktop.yaml"] = orig })
	if migrationRegistry["desktop.yaml"] == nil {
		migrationRegistry["desktop.yaml"] = map[int]migrationFunc{}
	}
	migrationRegistry["desktop.yaml"][1] = func(data []byte) ([]byte, error) {
		var m map[string]any
		if err := yaml.Unmarshal(data, &m); err != nil {
			return nil, err
		}
		m["version"] = 2
		if ui, ok := m["ui"].(map[string]any); ok {
			ui["animations"] = true
		}
		return yaml.Marshal(m)
	}
	// Pretend current is 2 for this migration path by calling migrate steps directly.
	in := []byte("version: 1\nsession:\n  name: i3_next\n  compositor: sway\npaths:\n  config_dir: x\n  data_dir: x\n  state_dir: x\n  cache_dir: x\nlogging:\n  level: info\n  format: text\nui:\n  shell: quickshell\n  theme: default\n")
	out := in
	var err error
	out, ok, err := migrateOne("desktop.yaml", 1, out)
	if err != nil || !ok {
		t.Fatalf("migrateOne: ok=%v err=%v", ok, err)
	}
	golden := filepath.Join("testdata", "golden", "desktop.v1-v2.yaml")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, out, 0o644); err != nil { //nolint:gosec // test file permissions
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("read golden (set UPDATE_GOLDEN=1 to create): %v", err)
	}
	if !bytes.Equal(out, want) {
		t.Fatalf("migration output mismatch\n--- got ---\n%s\n--- want ---\n%s", out, want)
	}
	if !strings.Contains(string(out), "version: 2") {
		t.Fatalf("expected version 2 in output: %s", out)
	}
	if !strings.Contains(string(out), "animations: true") {
		t.Fatalf("expected migrated field in output: %s", out)
	}
}

func TestMigrateBytesNoPath(t *testing.T) {
	// Ensure no migration is registered for this path.
	oldReg := migrationRegistry["desktop.yaml"]
	t.Cleanup(func() { migrationRegistry["desktop.yaml"] = oldReg })
	migrationRegistry["desktop.yaml"] = map[int]migrationFunc{}

	_, changed, issues := migrateIfNeeded("desktop.yaml", filepath.Join(t.TempDir(), "desktop.yaml"), []byte("version: 1\nsession:\n  name: x\n"), 2)
	if changed {
		t.Fatal("expected no change without migration path")
	}
	found := false
	for _, is := range issues {
		if is.Err != nil && strings.Contains(is.Err.Error(), "unsupported config version") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected unsupported-version issue, got %v", issues)
	}
}

func TestMigrateFileCreatesBackupBeforeWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "desktop.yaml")
	orig := []byte("version: 1\nsession:\n  name: orig\n  compositor: sway\npaths:\n  config_dir: x\n  data_dir: x\n  state_dir: x\n  cache_dir: x\nlogging:\n  level: info\n  format: text\nui:\n  shell: quickshell\n  theme: default\n")
	if err := os.WriteFile(path, orig, 0o644); err != nil { //nolint:gosec // test file permissions
		t.Fatal(err)
	}

	oldReg := migrationRegistry["desktop.yaml"]
	t.Cleanup(func() { migrationRegistry["desktop.yaml"] = oldReg })
	migrationRegistry["desktop.yaml"] = map[int]migrationFunc{
		1: func(data []byte) ([]byte, error) {
			var m map[string]any
			if err := yaml.Unmarshal(data, &m); err != nil {
				return nil, err
			}
			m["version"] = 2
			return yaml.Marshal(m)
		},
	}

	out, changed, issues := migrateIfNeeded("desktop.yaml", path, orig, 2)
	if !changed {
		t.Fatalf("expected migration change, issues=%v", issues)
	}
	if !strings.Contains(string(out), "version: 2") {
		t.Fatalf("migrated output missing version 2: %s", out)
	}
	// Original file should now be migrated (atomic write).
	cur, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cur), "version: 2") {
		t.Fatalf("file not migrated: %s", cur)
	}
	// Backup must exist and contain original content.
	matches, err := filepath.Glob(filepath.Join(dir, "desktop.yaml.bak.*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("expected backup file")
	}
	bak, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bak, orig) {
		t.Fatalf("backup content mismatch:\n--- bak ---\n%s\n--- orig ---\n%s", bak, orig)
	}
	info, err := os.Stat(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("backup mode = %o, want 644", info.Mode().Perm())
	}
	// Migration reported backup creation.
	foundNote := false
	for _, is := range issues {
		if is.Err != nil && strings.Contains(is.Err.Error(), "backup created") {
			foundNote = true
		}
	}
	if !foundNote {
		t.Fatalf("expected backup-created issue note, got %v", issues)
	}
}
