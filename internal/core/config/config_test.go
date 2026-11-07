package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDesktopValid(t *testing.T) {
	b, _ := os.ReadFile("../../../configs/defaults/desktop.yaml")
	c, err := ParseDesktop(b)
	if err != nil {
		t.Fatal(err)
	}
	if c.Version != 1 || c.Session.Compositor != "sway" {
		t.Fatalf("bad parse: %+v", c)
	}
}

func TestParseUnknownField(t *testing.T) {
	data := []byte("version: 1\nfoo: bar\n")
	if _, err := ParseDesktop(data); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseAllFiles(t *testing.T) {
	cases := map[string]func([]byte) error{
		"plugins.yaml":   func(b []byte) error { _, e := ParsePlugins(b); return e },
		"services.yaml":  func(b []byte) error { _, e := ParseServices(b); return e },
		"shortcuts.yaml": func(b []byte) error { _, e := ParseShortcuts(b); return e },
	}
	for name, fn := range cases {
		b, _ := os.ReadFile(filepath.Join("..", "..", "..", "configs", "defaults", name))
		if err := fn(b); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}
