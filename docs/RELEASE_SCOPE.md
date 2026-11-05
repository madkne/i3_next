# i3_next v1 Release Scope

Status: Draft v1.0 (scope freeze candidate)
Target version: 1.0.0
Last updated: 2026-11-05

## 1. Purpose

This document freezes the exact scope of i3_next v1. Anything not listed as
in-scope here is treated as backlog for a later release, even if it appears in
the README or in the issue tracker.

## 2. Product decision (session model)

v1 ships a **complete Sway-based desktop session** (display-manager entry +
compositor + core daemon + Quickshell shell), not merely a shell running on top
of an existing session. The session is launched from a Wayland session desktop
entry; Sway, i3-nextd and Quickshell are started in a defined order by
`i3-next-session`.

## 3. In-scope (v1 Must-have)

### 3.1 Core runtime

- Daemon lifecycle (start/stop/health/reload), single instance per session
- Process supervisor with restart policy and backoff
- Internal event bus and runtime state
- Structured logging, diagnostics (`i3-next-doctor`), `i3-nextctl`

### 3.2 Configuration

- Versioned schemas for `desktop.yaml`, `services.yaml`, `plugins.yaml`,
  `shortcuts.yaml`
- XDG-compliant paths, defaults, validation, atomic writes, migration, backup

### 3.3 Sway integration

- Sway IPC client + adapter (commands, queries, subscriptions, reconnect)
- Window/workspace/output/input models and events
- Generated, validated Sway config from templates (no silent overwrite)

### 3.4 Core/Shell API

- Unix domain socket + JSON-RPC, versioned, permission-restricted
- Status, config, service, workspace, output, notification, launcher APIs

### 3.5 Desktop shell (Quickshell/QML)

- Panel (clock, workspaces, network, audio, battery)
- Application launcher (desktop entries, safe execution)
- Notification popups + Notification Center
- Control Center (audio, brightness, network, Bluetooth where supported)
- OSD (volume, brightness, workspace change)
- Session status + logout/restart/shutdown
- Multi-monitor panel lifecycle, reconnect to daemon

### 3.6 System services (Must-have)

- Session manager, wallpaper manager, notification service
- Audio (PipeWire/WirePlumber), network status, power/battery
- Idle manager + lock screen (secure, tested tooling)
- Screenshot, clipboard integration (Wayland-supported scope)
- Display (outputs, resolution, scale), media controls (MPRIS)

### 3.7 Session, packaging, docs, tests

- Display-manager desktop entry, environment setup, recovery paths
- Debian/Ubuntu packages, install/uninstall/upgrade preserving user data
- Documentation, CI, diagnostics, manual QA checklist

## 4. Out-of-scope (backlog for later releases)

Explicitly NOT in v1:

- Third-party plugin execution with strong sandbox (only built-ins allowed;
  plugin host may exist but third-party loading disabled by default)
- Cloud sync of settings
- Advanced multi-user / multi-session scenarios
- Bluetooth full manager (beyond status where trivial)
- File search and advanced command palette
- Persistent searchable clipboard history
- Advanced layout manager, color picker, advanced system monitor
- Workspace overview and advanced animations
- Power profiles and vendor-specific tuning
- ARM64 as a release-gated platform (see section 6)
- Arch Linux package as a release-gated artifact (best-effort only)

## 5. Target distributions

Tier 1 (release-gated, must install and pass release gate on a clean system):

| Distribution | Version | Architecture | Notes |
|---|---|---|---|
| Ubuntu LTS | 24.04 (Noble) | amd64 | Primary reference platform |
| Debian Stable | 13 (Trixie) | amd64 | Primary reference platform |

Tier 2 (best-effort, not release-gated):

| Distribution | Version | Notes |
|---|---|---|
| Ubuntu LTS | 22.04 (Jammy) | Older Sway/Qt; may need backports |
| Debian Stable | 12 (Bookworm) | Older Sway/Qt; best-effort |
| Arch Linux | rolling | Community/PKGBUILD, not release-gated |
| Fedora | current | Best-effort, not tested in CI |

All other distributions are unsupported.

## 6. Hardware architecture

- Minimum and release-gated: **x86_64 (amd64)**.
- ARM64 (aarch64): **not release-gated in v1**. Status: experimental /
  best-effort. No prebuilt artifacts or CI coverage are promised; builds may
  work but are not validated. Full ARM64 support is targeted for a later
  release pending hardware access and testing.
- 32-bit (i386, armhf) and other architectures: unsupported.

Minimum hardware (x86_64):

- CPU: 2 cores
- RAM: 4 GiB
- Disk: 2 GiB free for installation + 1 GiB runtime/config
- Display: at least one output supported by Sway

## 7. Compatibility matrix

Versions are minimum supported unless stated. "Tested" reflects the versions
used in development/CI on the Tier 1 platforms.

| Component | Minimum | Recommended | Tested (dev) | Notes |
|---|---|---|---|---|
| Go (toolchain) | 1.24 | 1.24/1.25 | 1.27.3 (dev) | `go.mod` currently declares 1.27.3; must be corrected to a stable toolchain before release |
| Sway | 1.8 | 1.10+ | 1.12 | Feature detection required for version differences |
| Wayland (protocol/libs) | 1.21 | 1.22+ | 1.26.0 | Compositor/protocol baseline |
| Quickshell | 0.2 | 0.3+ | 0.3.2 | May require backport/PPA on Tier 1 older releases |
| Qt / QML | 6.4 | 6.6+ | 6.11.2 | Qt 6 only; Qt 5 unsupported |
| PipeWire | 0.3 | 1.0+ | distro default | Optional but expected for audio service |
| NetworkManager | 1.40 | latest | distro default | Optional dependency for network management |
| D-Bus | session+system bus | - | distro default | Required |

## 8. Mandatory vs optional dependencies

Mandatory: Linux kernel with KMS/DRM, Sway, Wayland compositor libs, Go-built
binaries, D-Bus (session + system), Qt 6 runtime (for Quickshell), XDG dirs.

Optional (degraded mode, not crash): PipeWire/WirePlumber, NetworkManager,
Bluetooth stack, MPRIS players, upower, swaylock/swayidle, grim/slurp,
wl-clipboard. Absence of an optional dependency must move the related service
to a degraded state, never take down the whole desktop.

## 9. Acceptance criteria (v1 release gate)

v1 is releasable only when all of the following hold:

- Clean install succeeds on at least one clean system per Tier 1 distribution
- User can log in via display manager into the i3_next session
- Sway, daemon and Quickshell start without manual intervention
- Panel and launcher are usable
- Basic window and workspace management works
- Notifications are displayed
- Audio, network and battery status (where hardware present) reported correctly
- Lock, logout, reboot, shutdown follow a safe and reliable path
- Backend disconnect does not permanently crash the UI; reconnect works
- Corrupt config does not lose the session or cause restart loops
- Uninstall and upgrade do not remove user data without consent
- CI (format, lint, vet, unit tests, build) passes
- No critical known vulnerabilities in dependencies or code
- Install, troubleshooting and recovery docs published
- Version, commit, build date and changelog identifiable in artifacts
- Release candidate tested on real hardware with results recorded
- Known issues and bug reporting path published

## 10. Versioning and compatibility policy

- Versioning: SemVer (`MAJOR.MINOR.PATCH`). v1 = `1.0.x`.
- Config schemas are versioned; migrations run on load; invalid config never
  crashes the daemon.
- API between core and shell is versioned; mismatch is detected and reported.
- Backward compatibility for config is maintained within v1.x; breaking
  changes require a major version bump and a documented migration.

## 11. Known limitations (v1)

- Third-party plugins disabled by default.
- ARM64 not release-gated.
- Some optional services may be degraded on minimal installs.
- No cloud sync, no advanced multi-session.
