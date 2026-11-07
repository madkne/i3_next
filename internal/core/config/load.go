package config

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Layer describes a configuration source in precedence order.
//
// The documented and implemented order is:
//  1. defaults (embedded)
//  2. /etc/i3-next (system)
//  3. ~/.config/i3-next (user, from ConfigDir())
//  4. environment variables (I3NEXT_*)
//  5. CLI flags (highest precedence)
//
// Later layers override earlier layers. Mapping keys are merged recursively;
// scalars and sequences replace the value from earlier layers.
type Layer struct {
	Name string
	Path string // directory containing config files; empty for non-file layers
}

// DefaultLayers returns the file-based configuration layers in precedence order,
// excluding defaults, environment, and flags.
func DefaultLayers() []Layer {
	return []Layer{
		{Name: "system", Path: filepath.Join(string(filepath.Separator), "etc", "i3-next")},
		{Name: "user", Path: ConfigDir()},
	}
}

// FileNames lists the managed configuration files.
var FileNames = []string{"desktop.yaml", "plugins.yaml", "services.yaml", "shortcuts.yaml"}

// LoadOptions controls layered loading.
type LoadOptions struct {
	// Layers are file-based layers applied after embedded defaults, in order.
	// If nil, DefaultLayers() is used.
	Layers []Layer
	// Env, when true, applies I3NEXT_* environment variables after file layers.
	Env bool
	// Flags applies explicit CLI overrides after environment variables.
	// Keys are dotted paths (for example "logging.level", "ui.theme").
	Flags map[string]string
}

// MergeYAML merges overlay into base and returns the merged YAML document.
// Both inputs must be YAML mappings (or empty). Overlay mapping keys replace or
// recursively merge into base mapping keys; non-mapping values replace base.
func MergeYAML(base, overlay []byte) ([]byte, error) {
	var b yaml.Node
	if len(bytes.TrimSpace(base)) > 0 {
		if err := yaml.Unmarshal(base, &b); err != nil {
			return nil, fmt.Errorf("decode base: %w", err)
		}
	}
	var o yaml.Node
	if len(bytes.TrimSpace(overlay)) > 0 {
		if err := yaml.Unmarshal(overlay, &o); err != nil {
			return nil, fmt.Errorf("decode overlay: %w", err)
		}
	}
	merged, err := mergeNode(documentRoot(&b), documentRoot(&o))
	if err != nil {
		return nil, err
	}
	if merged == nil {
		return []byte{}, nil
	}
	out, err := yaml.Marshal(merged)
	if err != nil {
		return nil, fmt.Errorf("encode merged: %w", err)
	}
	return out, nil
}

func documentRoot(n *yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		return n.Content[0]
	}
	if n.Kind == 0 {
		return nil
	}
	return n
}

func mergeNode(base, overlay *yaml.Node) (*yaml.Node, error) {
	if overlay == nil {
		return cloneNode(base), nil
	}
	if base == nil {
		return cloneNode(overlay), nil
	}
	if base.Kind == yaml.MappingNode && overlay.Kind == yaml.MappingNode {
		return mergeMapping(base, overlay)
	}
	return cloneNode(overlay), nil
}

func mergeMapping(base, overlay *yaml.Node) (*yaml.Node, error) {
	out := &yaml.Node{Kind: yaml.MappingNode, Tag: base.Tag}
	index := map[string]int{}
	for i := 0; i+1 < len(base.Content); i += 2 {
		k := base.Content[i].Value
		index[k] = len(out.Content)
		out.Content = append(out.Content, cloneNode(base.Content[i]), cloneNode(base.Content[i+1]))
	}
	for i := 0; i+1 < len(overlay.Content); i += 2 {
		k := overlay.Content[i]
		v := overlay.Content[i+1]
		if pos, ok := index[k.Value]; ok {
			merged, err := mergeNode(out.Content[pos+1], v)
			if err != nil {
				return nil, err
			}
			out.Content[pos+1] = merged
			continue
		}
		index[k.Value] = len(out.Content)
		out.Content = append(out.Content, cloneNode(k), cloneNode(v))
	}
	return out, nil
}

func cloneNode(n *yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}
	c := *n
	if len(n.Content) > 0 {
		c.Content = make([]*yaml.Node, len(n.Content))
		for i, child := range n.Content {
			c.Content[i] = cloneNode(child)
		}
	}
	if n.Alias != nil {
		c.Alias = cloneNode(n.Alias)
	}
	return &c
}

// LoadDesktopMerged loads desktop.yaml by applying layers in order:
// embedded defaults, configured file layers, environment (if opts.Env), and flags.
func LoadDesktopMerged(opts LoadOptions) (DesktopConfig, []byte, error) {
	b, err := LoadMergedBytes("desktop.yaml", DefaultDesktop, opts)
	if err != nil {
		return DesktopConfig{}, nil, err
	}
	c, err := ParseDesktop(b)
	if err != nil {
		return DesktopConfig{}, nil, err
	}
	return c, b, nil
}

// LoadMergedBytes loads and merges a named config file across layers.
func LoadMergedBytes(name string, defaults []byte, opts LoadOptions) ([]byte, error) {
	layers := opts.Layers
	if layers == nil {
		layers = DefaultLayers()
	}
	merged := append([]byte(nil), defaults...)
	for _, l := range layers {
		if l.Path == "" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(l.Path, name))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s layer %s: %w", name, l.Name, err)
		}
		merged, err = MergeYAML(merged, b)
		if err != nil {
			return nil, fmt.Errorf("merge %s layer %s: %w", name, l.Name, err)
		}
	}
	if opts.Env {
		envOverlay, err := EnvOverlay(name)
		if err != nil {
			return nil, err
		}
		if len(envOverlay) > 0 {
			merged, err = MergeYAML(merged, envOverlay)
			if err != nil {
				return nil, fmt.Errorf("merge %s env: %w", name, err)
			}
		}
	}
	if len(opts.Flags) > 0 {
		flagOverlay, err := FlagsOverlay(name, opts.Flags)
		if err != nil {
			return nil, err
		}
		if len(flagOverlay) > 0 {
			merged, err = MergeYAML(merged, flagOverlay)
			if err != nil {
				return nil, fmt.Errorf("merge %s flags: %w", name, err)
			}
		}
	}
	return merged, nil
}

// EnvOverlay builds a YAML overlay from I3NEXT_* environment variables relevant
// to the named config file.
//
// Known desktop mappings (YAML keys may contain underscores):
//
//	I3NEXT_DESKTOP_SESSION_NAME       -> session.name
//	I3NEXT_DESKTOP_SESSION_COMPOSITOR -> session.compositor
//	I3NEXT_DESKTOP_LOGGING_LEVEL      -> logging.level
//	I3NEXT_DESKTOP_LOGGING_FORMAT     -> logging.format
//	I3NEXT_DESKTOP_UI_SHELL           -> ui.shell
//	I3NEXT_DESKTOP_UI_THEME           -> ui.theme
//	I3NEXT_DESKTOP_PATHS_CONFIG_DIR   -> paths.config_dir
//	I3NEXT_DESKTOP_PATHS_DATA_DIR     -> paths.data_dir
//	I3NEXT_DESKTOP_PATHS_STATE_DIR    -> paths.state_dir
//	I3NEXT_DESKTOP_PATHS_CACHE_DIR    -> paths.cache_dir
//	I3NEXT_DESKTOP_VERSION            -> version
//
// For other files, or unknown keys, the suffix after the file prefix is
// lowercased and split on '_' into path segments.
func EnvOverlay(name string) ([]byte, error) {
	prefix, ok := envPrefixFor(name)
	if !ok {
		return nil, nil
	}
	root := map[string]any{}
	for _, kv := range os.Environ() {
		k, v, ok := strings.Cut(kv, "=")
		if !ok || !strings.HasPrefix(k, prefix) {
			continue
		}
		suffix := strings.TrimPrefix(k, prefix)
		if suffix == "" {
			continue
		}
		path := envPath(name, suffix)
		if len(path) == 0 {
			continue
		}
		if err := setNested(root, path, parseScalar(v)); err != nil {
			return nil, fmt.Errorf("env %s: %w", k, err)
		}
	}
	if len(root) == 0 {
		return nil, nil
	}
	return yaml.Marshal(root)
}

func envPrefixFor(name string) (string, bool) {
	switch name {
	case "desktop.yaml":
		return "I3NEXT_DESKTOP_", true
	case "plugins.yaml":
		return "I3NEXT_PLUGINS_", true
	case "services.yaml":
		return "I3NEXT_SERVICES_", true
	case "shortcuts.yaml":
		return "I3NEXT_SHORTCUTS_", true
	default:
		return "", false
	}
}

var desktopEnvPaths = map[string][]string{
	"SESSION_NAME":       {"session", "name"},
	"SESSION_COMPOSITOR": {"session", "compositor"},
	"LOGGING_LEVEL":      {"logging", "level"},
	"LOGGING_FORMAT":     {"logging", "format"},
	"UI_SHELL":           {"ui", "shell"},
	"UI_THEME":           {"ui", "theme"},
	"PATHS_CONFIG_DIR":   {"paths", "config_dir"},
	"PATHS_DATA_DIR":     {"paths", "data_dir"},
	"PATHS_STATE_DIR":    {"paths", "state_dir"},
	"PATHS_CACHE_DIR":    {"paths", "cache_dir"},
	"VERSION":            {"version"},
}

func envPath(name, suffix string) []string {
	if name == "desktop.yaml" {
		if p, ok := desktopEnvPaths[suffix]; ok {
			out := make([]string, len(p))
			copy(out, p)
			return out
		}
	}
	parts := strings.Split(strings.ToLower(suffix), "_")
	out := parts[:0]
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// FlagsOverlay builds a YAML overlay from dotted-path flag overrides.
// Keys may optionally be prefixed with the file stem (for example
// "desktop.logging.level" or "logging.level").
func FlagsOverlay(name string, flags map[string]string) ([]byte, error) {
	if len(flags) == 0 {
		return nil, nil
	}
	stem := strings.TrimSuffix(name, ".yaml")
	root := map[string]any{}
	for k, v := range flags {
		path := strings.Split(k, ".")
		if len(path) > 0 && path[0] == stem {
			path = path[1:]
		}
		if len(path) == 0 {
			continue
		}
		clean := path[:0]
		for _, p := range path {
			if p != "" {
				clean = append(clean, p)
			}
		}
		if len(clean) == 0 {
			continue
		}
		if err := setNested(root, clean, parseScalar(v)); err != nil {
			return nil, fmt.Errorf("flag %s: %w", k, err)
		}
	}
	if len(root) == 0 {
		return nil, nil
	}
	return yaml.Marshal(root)
}

func setNested(root map[string]any, path []string, value any) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}
	cur := root
	for _, part := range path[:len(path)-1] {
		if part == "" {
			return fmt.Errorf("empty path segment")
		}
		next, ok := cur[part]
		if !ok {
			child := map[string]any{}
			cur[part] = child
			cur = child
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			child = map[string]any{}
			cur[part] = child
		}
		cur = child
	}
	leaf := path[len(path)-1]
	if leaf == "" {
		return fmt.Errorf("empty path segment")
	}
	cur[leaf] = value
	return nil
}

func parseScalar(v string) any {
	var out any
	if err := yaml.Unmarshal([]byte(v), &out); err == nil {
		return out
	}
	return v
}

// InstallDefaults writes embedded default config files into dir. Files are
// created with O_EXCL and are never overwritten; existing files are left
// untouched. The directory is created with mode 0700 if needed.
func InstallDefaults(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	files := map[string][]byte{
		"desktop.yaml":   DefaultDesktop,
		"plugins.yaml":   DefaultPlugins,
		"services.yaml":  DefaultServices,
		"shortcuts.yaml": DefaultShortcuts,
	}
	for _, name := range FileNames {
		if err := writeDefaultFile(filepath.Join(dir, name), files[name]); err != nil {
			return err
		}
	}
	return nil
}

func writeDefaultFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}
		return fmt.Errorf("create %s: %w", filepath.Base(path), err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("write %s: %w", filepath.Base(path), err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("sync %s: %w", filepath.Base(path), err)
	}
	return nil
}

// InstallUserDefaults installs defaults into ConfigDir().
func InstallUserDefaults() error {
	return InstallDefaults(ConfigDir())
}

var _ fs.FS
