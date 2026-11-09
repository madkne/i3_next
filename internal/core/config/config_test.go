package config

import (
	"testing"
)

func TestParseDesktopValid(t *testing.T) {
	c, err := ParseDesktop(DefaultDesktop)
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
		"desktop.yaml":   func(b []byte) error { _, e := ParseDesktop(b); return e },
		"plugins.yaml":   func(b []byte) error { _, e := ParsePlugins(b); return e },
		"services.yaml":  func(b []byte) error { _, e := ParseServices(b); return e },
		"shortcuts.yaml": func(b []byte) error { _, e := ParseShortcuts(b); return e },
	}
	data := map[string][]byte{
		"desktop.yaml":   DefaultDesktop,
		"plugins.yaml":   DefaultPlugins,
		"services.yaml":  DefaultServices,
		"shortcuts.yaml": DefaultShortcuts,
	}
	for name, fn := range cases {
		if err := fn(data[name]); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
}
