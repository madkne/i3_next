package config

import _ "embed"

//go:embed defaults/desktop.yaml
var DefaultDesktop []byte

//go:embed defaults/plugins.yaml
var DefaultPlugins []byte

//go:embed defaults/services.yaml
var DefaultServices []byte

//go:embed defaults/shortcuts.yaml
var DefaultShortcuts []byte
