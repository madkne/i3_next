package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"i3_next/internal/core/config"
	"i3_next/internal/version"
)

type check struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Detail string `json:"detail"`
	Remedy string `json:"remedy,omitempty"`
}

func main() {
	showVersion := flag.Bool("version", false, "Show version")
	jsonOut := flag.Bool("json", false, "Output results as JSON")
	strict := flag.Bool("strict", false, "Treat warnings as failures")
	flag.Parse()

	if *showVersion {
		fmt.Printf("i3-next-doctor %s (%s, %s)\n", version.Version, version.Commit, version.BuildDate)
		return
	}

	checks := runChecks()

	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(checks); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	} else {
		for _, c := range checks {
			fmt.Printf("[%s] %s\n", strings.ToUpper(c.Status), c.Detail)
			if c.Remedy != "" && c.Status != "ok" {
				fmt.Printf("       remedy: %s\n", c.Remedy)
			}
		}
	}

	os.Exit(exitCode(checks, *strict))
}

func exitCode(checks []check, strict bool) int {
	hasFail := false
	hasWarn := false
	for _, c := range checks {
		switch c.Status {
		case "fail":
			hasFail = true
		case "warn":
			hasWarn = true
		}
	}
	if hasFail {
		return 2
	}
	if strict && hasWarn {
		return 1
	}
	return 0
}

func runChecks() []check {
	var checks []check

	// Sway (required)
	if p, err := exec.LookPath("sway"); err != nil {
		checks = append(checks, check{"sway", "fail", "sway not found", "Install sway (apt install sway)."})
	} else {
		detail := "sway installed"
		if out, err := exec.Command("sway", "--version").CombinedOutput(); err == nil {
			detail = "sway installed (" + strings.TrimSpace(string(out)) + ")"
		}
		_ = p
		checks = append(checks, check{"sway", "ok", detail, ""})
	}

	// Quickshell (optional but expected)
	if _, err := exec.LookPath("quickshell"); err != nil {
		checks = append(checks, check{"quickshell", "warn", "quickshell not found", "Install quickshell or build from source."})
	} else {
		detail := "quickshell installed"
		if out, err := exec.Command("quickshell", "--version").CombinedOutput(); err == nil {
			detail = "quickshell installed (" + strings.TrimSpace(string(out)) + ")"
		}
		checks = append(checks, check{"quickshell", "ok", detail, ""})
	}

	// qmllint / Qt6
	if _, err := exec.LookPath("qmllint"); err != nil {
		checks = append(checks, check{"qt6-qmllint", "warn", "qmllint not found", "Install qt6-declarative-dev-tools."})
	} else {
		checks = append(checks, check{"qt6-qmllint", "ok", "qmllint installed", ""})
	}

	// Wayland environment
	if os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland" {
		checks = append(checks, check{"wayland-env", "ok", "Wayland session detected", ""})
	} else {
		checks = append(checks, check{"wayland-env", "warn", "No Wayland environment detected", "Run inside a Wayland/Sway session."})
	}

	// D-Bus session bus
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") != "" {
		checks = append(checks, check{"dbus-session", "ok", "session bus available", ""})
	} else {
		checks = append(checks, check{"dbus-session", "warn", "DBUS_SESSION_BUS_ADDRESS not set", "Run dbus-update-activation-environment."})
	}

	// D-Bus system bus
	if _, err := os.Stat("/var/run/dbus/system_bus_socket"); err == nil {
		checks = append(checks, check{"dbus-system", "ok", "system bus socket present", ""})
	} else {
		checks = append(checks, check{"dbus-system", "warn", "system bus socket missing", "Start dbus system service."})
	}

	// Optional tools
	optional := []string{"swaylock", "swayidle", "swaybg", "grim", "slurp", "wl-copy", "wpctl", "brightnessctl"}
	for _, t := range optional {
		if _, err := exec.LookPath(t); err != nil {
			checks = append(checks, check{"tool-" + t, "warn", t + " not found", "Install " + t + " for full functionality."})
		} else {
			checks = append(checks, check{"tool-" + t, "ok", t + " installed", ""})
		}
	}

	// Directories
	for _, d := range []string{config.ConfigDir(), config.StateDir(), config.DataDir(), config.CacheDir(), config.RuntimeDir()} {
		if info, err := os.Stat(d); err == nil {
			if info.Mode().Perm()&0o077 != 0 {
				checks = append(checks, check{"dir-" + filepath.Base(d), "warn", d + " permissions too open (" + info.Mode().Perm().String() + ")", "chmod 700 " + d})
			} else {
				checks = append(checks, check{"dir-" + filepath.Base(d), "ok", d + " present", ""})
			}
		} else {
			checks = append(checks, check{"dir-" + filepath.Base(d), "warn", d + " missing", "Run i3-nextd once to create runtime directories."})
		}
	}

	// Config file (best effort)
	cfgPath := filepath.Join(config.ConfigDir(), "desktop.yaml")
	if _, err := os.Stat(cfgPath); err != nil {
		checks = append(checks, check{"config", "warn", "desktop.yaml not found", "Run i3-nextd to install defaults."})
	} else {
		checks = append(checks, check{"config", "ok", "desktop.yaml present", ""})
	}

	// API socket
	sock := config.SocketPath()
	if _, err := os.Stat(sock); err != nil {
		checks = append(checks, check{"api-socket", "warn", "daemon socket not present", "Start i3-nextd."})
	} else {
		checks = append(checks, check{"api-socket", "ok", "daemon socket present", ""})
	}

	return checks
}
