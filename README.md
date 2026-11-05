# i3_next

Modern Modular Wayland Desktop Environment.

i3_next is a Sway-based Linux Desktop Environment
with a Go-powered backend and a modern desktop shell.

## Architecture

- Sway: Wayland compositor
- Go: Desktop Core
- Quickshell: Desktop Shell
- QML: User Interface
- D-Bus: System Integration
- JSON-RPC: Internal API
- SQLite: Persistent Storage

## Planned Features (v1 scope — not yet implemented)

- Window Management
- Workspace Management
- Desktop Shell
- Notification Center
- Control Center
- Clipboard Manager
- Plugin System
- Configuration Management

## Development

1. init dependencies

```bash
sudo apt update

sudo apt install -y \
    git \
    curl \
    build-essential \
    pkg-config \
    cmake \
    ninja-build \
    make \
    jq \
    dbus \
    dbus-user-session \
    sway \
    swaybg \
    swayidle \
    swaylock \
    waybar \
    wl-clipboard \
    grim \
    slurp \
    pipewire \
    wireplumber \
    libwayland-dev \
    wayland-protocols \
    qemu-system-x86 libvirt-daemon-system virtinst virt-viewer genisoimage rsync

    sudo usermod -aG libvirt,kvm $USER

```

2. for testing on kvm

```bash
make vm-up

make vm-test
```

### Requirements

- Linux
- Go
- Git
- Sway
- Wayland

### Build

```bash
make build
```

## Structure

```text
i3_next/
│
├── cmd/
│   ├── i3-nextd/                 # Desktop Core Daemon
│   ├── i3-nextctl/               # CLI management tool
│   ├── i3-next-session/          # Session startup
│   ├── i3-next-doctor/           # Diagnostics and health checks
│   └── i3-next-plugin-host/      # Plugin execution host
│
├── internal/
│   │
│   ├── app/
│   │   ├── bootstrap/             # Application initialization
│   │   ├── lifecycle/             # Start, stop, restart
│   │   ├── supervisor/            # Process supervision
│   │   └── shutdown/              # Graceful shutdown
│   │
│   ├── core/
│   │   ├── eventbus/              # Internal event system
│   │   ├── config/                # Configuration management
│   │   ├── state/                 # Runtime state
│   │   ├── registry/              # Service registry
│   │   ├── permissions/           # Permission management
│   │   └── logging/               # Structured logging
│   │
│   ├── adapters/
│   │   ├── sway/
│   │   │   ├── client/            # Sway IPC client
│   │   │   ├── commands/          # Sway command execution
│   │   │   ├── events/            # Window/workspace/output events
│   │   │   ├── models/            # Sway data models
│   │   │   └── monitor/           # Connection monitoring
│   │   │
│   │   ├── wayland/
│   │   │   ├── outputs/
│   │   │   ├── protocols/
│   │   │   └── session/
│   │   │
│   │   ├── dbus/
│   │   │   ├── system/
│   │   │   └── session/
│   │   │
│   │   └── process/
│   │       ├── manager/
│   │       └── executor/
│   │
│   ├── services/
│   │   ├── window-manager/
│   │   ├── workspace/
│   │   ├── layout/
│   │   ├── session/
│   │   ├── launcher/
│   │   ├── notifications/
│   │   ├── clipboard/
│   │   ├── screenshot/
│   │   ├── wallpaper/
│   │   ├── display/
│   │   ├── audio/
│   │   ├── network/
│   │   ├── bluetooth/
│   │   ├── power/
│   │   ├── media/
│   │   ├── idle/
│   │   └── lockscreen/
│   │
│   ├── api/
│   │   ├── dbus/                  # Public D-Bus API
│   │   ├── rpc/                   # Unix socket JSON-RPC
│   │   ├── events/                # Event definitions
│   │   └── contracts/             # API contracts
│   │
│   ├── plugins/
│   │   ├── manager/
│   │   ├── loader/
│   │   ├── runtime/
│   │   ├── sandbox/
│   │   └── sdk/
│   │
│   ├── storage/
│   │   ├── sqlite/
│   │   ├── migrations/
│   │   └── repositories/
│   │
│   └── system/
│       ├── hardware/
│       ├── power/
│       ├── user/
│       └── environment/
│
├── ui/
│   └── quickshell/
│       ├── shell/
│       │   ├── Main.qml
│       │   ├── Shell.qml
│       │   └── Session.qml
│       │
│       ├── panels/
│       │   ├── TopBar/
│       │   ├── Dock/
│       │   ├── ControlCenter/
│       │   ├── NotificationCenter/
│       │   ├── WorkspaceOverview/
│       │   └── OSD/
│       │
│       ├── launchers/
│       │   ├── ApplicationLauncher/
│       │   ├── CommandPalette/
│       │   └── FileSearch/
│       │
│       ├── widgets/
│       │   ├── Clock/
│       │   ├── Network/
│       │   ├── Audio/
│       │   ├── Battery/
│       │   ├── Media/
│       │   └── SystemMonitor/
│       │
│       ├── components/
│       ├── services/
│       │   ├── Backend.qml
│       │   ├── EventBridge.qml
│       │   └── Settings.qml
│       │
│       ├── themes/
│       │   ├── default/
│       │   └── presets/
│       │
│       ├── settings/
│       ├── assets/
│       └── qmldir
│
├── tools/
│   ├── clipboard/
│   ├── screenshot/
│   ├── color-picker/
│   ├── layout-manager/
│   ├── workspace-manager/
│   ├── command-palette/
│   ├── file-search/
│   └── system-monitor/
│
├── plugins/
│   ├── examples/
│   └── built-in/
│
├── configs/
│   ├── defaults/
│   │   ├── desktop.yaml
│   │   ├── services.yaml
│   │   ├── plugins.yaml
│   │   └── shortcuts.yaml
│   │
│   ├── sway/
│   │   ├── config
│   │   ├── keybindings
│   │   ├── window-rules
│   │   └── outputs
│   │
│   └── themes/
│
├── packaging/
│   ├── debian/
│   ├── arch/
│   ├── systemd/
│   ├── dbus/
│   ├── wayland-sessions/
│   └── xdg/
│
├── scripts/
│   ├── dev-setup.sh
│   ├── build.sh
│   ├── install.sh
│   └── uninstall.sh
│
├── docs/
│   ├── architecture/
│   ├── api/
│   ├── plugins/
│   ├── development/
│   └── user-guide/
│
├── tests/
│   ├── integration/
│   ├── e2e/
│   └── fixtures/
│
├── Makefile
├── go.mod
├── go.sum
├── CMakeLists.txt
├── README.md
├── LICENSE
└── .gitignore
```
