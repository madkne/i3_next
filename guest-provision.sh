#!/usr/bin/env bash
# Runs inside the guest as root (called by vm.sh provision). Idempotent.
set -Eeuo pipefail
export DEBIAN_FRONTEND=noninteractive

DEV_USER="${DEV_USER:-dev}"
GO_VERSION="${GO_VERSION:-}"
INSTALL_QUICKSHELL="${INSTALL_QUICKSHELL:-1}"

log()  { printf '\033[1;34m[guest]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[guest warn]\033[0m %s\n' "$*"; }

log "installing packages"
apt-get update -qq
apt-get install -y \
  git curl ca-certificates build-essential pkg-config cmake ninja-build make \
  jq rsync file \
  dbus dbus-user-session libpam-systemd \
  sway swaybg swayidle swaylock waybar wl-clipboard grim slurp foot wtype wayvnc \
  pipewire wireplumber xdg-desktop-portal-wlr \
  libwayland-dev wayland-protocols libgl1-mesa-dri \
  qemu-guest-agent spice-vdagent fonts-dejavu-core

install_go() {
  if [ -z "$GO_VERSION" ]; then
    warn "go.mod version unknown, installing distro golang"
    apt-get install -y golang-go
    return
  fi
  local v="$GO_VERSION" arch
  if [[ "$v" =~ ^[0-9]+\.[0-9]+$ ]]; then v="$v.0"; fi
  if [ -x /usr/local/go/bin/go ] && /usr/local/go/bin/go version | grep -q "go$v "; then
    log "Go $v already installed"
    return
  fi
  case "$(uname -m)" in
    x86_64)  arch=amd64 ;;
    aarch64) arch=arm64 ;;
    *) warn "unsupported arch for Go download"; return 1 ;;
  esac
  log "installing Go $v"
  curl -fsSL "https://go.dev/dl/go${v}.linux-${arch}.tar.gz" -o /tmp/go.tgz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tgz
  rm -f /tmp/go.tgz
  ln -sf /usr/local/go/bin/go /usr/local/bin/go
  ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
}
install_go

if [ "$INSTALL_QUICKSHELL" = 1 ]; then
  apt-get install -y quickshell ||
    warn "quickshell is not available in this distro's apt; install it manually or build from source"
fi

log "writing session environment"
cat > /etc/i3-next-test.env <<'ENV'
# Software rendering is the most robust choice on a virtual GPU.
# Remove WLR_RENDERER when running with VM_GL=1 on the host.
WLR_RENDERER=pixman
WLR_NO_HARDWARE_CURSORS=1
XDG_SESSION_TYPE=wayland
XDG_CURRENT_DESKTOP=sway
ENV

mkdir -p /etc/systemd/system/getty@tty1.service.d
cat > /etc/systemd/system/getty@tty1.service.d/autologin.conf <<UNIT
[Service]
ExecStart=
ExecStart=-/sbin/agetty --autologin ${DEV_USER} --noclear %I \$TERM
UNIT
systemctl daemon-reload

PROFILE="/home/${DEV_USER}/.profile"
if ! grep -q '>>> i3-next-test >>>' "$PROFILE" 2>/dev/null; then
  cat >> "$PROFILE" <<'PROF'

# >>> i3-next-test >>>
# Start the real Sway session on tty1 when "vm.sh gui on" created the flag file.
if [ -z "${WAYLAND_DISPLAY:-}" ] && [ "$(tty 2>/dev/null)" = /dev/tty1 ] && [ -f /etc/i3-next-gui-session ]; then
  set -a; . /etc/i3-next-test.env; set +a
  cfg="$HOME/i3-next/configs/sway/config"
  # The repo config may still be empty: fall back to Sway's default one.
  if [ -s "$cfg" ]; then exec sway -c "$cfg"; else exec sway; fi
fi
# <<< i3-next-test <<<
PROF
fi

loginctl enable-linger "$DEV_USER"
systemctl start qemu-guest-agent 2>/dev/null || true

mkdir -p /var/lib/i3-next-vm
date -u +%FT%TZ > /var/lib/i3-next-vm/provisioned
log "provisioning done"
