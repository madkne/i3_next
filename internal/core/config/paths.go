package config

import (
	"os"
	"path/filepath"
)

const (
	AppName = "i3_next"

	BinaryDaemon     = "i3-nextd"
	BinaryControl    = "i3-nextctl"
	BinarySession    = "i3-next-session"
	BinaryDoctor     = "i3-next-doctor"
	BinaryPluginHost = "i3-next-plugin-host"

	RuntimeDirName = "i3-next"
	ConfigDirName  = "i3-next"
	DataDirName    = "i3-next"
	StateDirName   = "i3-next"
	CacheDirName   = "i3-next"

	SocketName   = "i3-next.sock"
	DatabaseName = "i3-next.db"
)

func xdgDir(envKey, fallback string) string {
	if v := os.Getenv(envKey); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, fallback)
}

func RuntimeDir() string {
	if v := os.Getenv("XDG_RUNTIME_DIR"); v != "" {
		return filepath.Join(v, RuntimeDirName)
	}
	return filepath.Join(os.TempDir(), RuntimeDirName)
}

func ConfigDir() string {
	return filepath.Join(xdgDir("XDG_CONFIG_HOME", ".config"), ConfigDirName)
}

func DataDir() string {
	return filepath.Join(xdgDir("XDG_DATA_HOME", ".local/share"), DataDirName)
}

func StateDir() string {
	return filepath.Join(xdgDir("XDG_STATE_HOME", ".local/state"), StateDirName)
}

func CacheDir() string {
	return filepath.Join(xdgDir("XDG_CACHE_HOME", ".cache"), CacheDirName)
}

func SocketPath() string {
	return filepath.Join(RuntimeDir(), SocketName)
}

func DatabasePath() string {
	return filepath.Join(StateDir(), DatabaseName)
}
