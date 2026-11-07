package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type DesktopConfig struct {
	Version int `yaml:"version"`
	Session struct {
		Name       string `yaml:"name"`
		Compositor string `yaml:"compositor"`
	} `yaml:"session"`
	Paths struct {
		ConfigDir string `yaml:"config_dir"`
		DataDir   string `yaml:"data_dir"`
		StateDir  string `yaml:"state_dir"`
		CacheDir  string `yaml:"cache_dir"`
	} `yaml:"paths"`
	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logging"`
	UI struct {
		Shell string `yaml:"shell"`
		Theme string `yaml:"theme"`
	} `yaml:"ui"`
}

type PluginEntry struct {
	Name    string `yaml:"name"`
	Enabled bool   `yaml:"enabled"`
}

type PluginsConfig struct {
	Version int           `yaml:"version"`
	Plugins []PluginEntry `yaml:"plugins"`
}

type ServiceEntry struct {
	Enabled   bool     `yaml:"enabled"`
	Required  bool     `yaml:"required"`
	DependsOn []string `yaml:"depends_on"`
}

type ServicesConfig struct {
	Version  int                     `yaml:"version"`
	Services map[string]ServiceEntry `yaml:"services"`
}

type Shortcut struct {
	Name    string `yaml:"name"`
	Binding string `yaml:"binding"`
	Command string `yaml:"command"`
}

type ShortcutsConfig struct {
	Version   int        `yaml:"version"`
	Shortcuts []Shortcut `yaml:"shortcuts"`
}

func strictUnmarshal(data []byte, v any) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

func ParseDesktop(data []byte) (DesktopConfig, error) {
	var c DesktopConfig
	if err := strictUnmarshal(data, &c); err != nil {
		return DesktopConfig{}, fmt.Errorf("desktop config: %w", err)
	}
	return c, nil
}

func ParsePlugins(data []byte) (PluginsConfig, error) {
	var c PluginsConfig
	if err := strictUnmarshal(data, &c); err != nil {
		return PluginsConfig{}, fmt.Errorf("plugins config: %w", err)
	}
	return c, nil
}

func ParseServices(data []byte) (ServicesConfig, error) {
	var c ServicesConfig
	if err := strictUnmarshal(data, &c); err != nil {
		return ServicesConfig{}, fmt.Errorf("services config: %w", err)
	}
	return c, nil
}

func ParseShortcuts(data []byte) (ShortcutsConfig, error) {
	var c ShortcutsConfig
	if err := strictUnmarshal(data, &c); err != nil {
		return ShortcutsConfig{}, fmt.Errorf("shortcuts config: %w", err)
	}
	return c, nil
}

func LoadDesktop(path string) (DesktopConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return DesktopConfig{}, err
	}
	return ParseDesktop(b)
}
