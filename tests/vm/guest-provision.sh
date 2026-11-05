#!/usr/bin/env bash
# Runs inside the guest as root (called by "vm.sh provision"). Idempotent.
#
# Logging:
#   * everything is timestamped and also written to /var/log/i3-next-provision.log
#   * every step reports its duration; failures print line, command and exit code
#   * PROVISION_DEBUG=1 enables full shell tracing (set -x)
set -Eeuo pipefail
export DEBIAN_FRONTEND=noninteractive

DEV_USER="${DEV_USER:-dev}"
GO_VERSION="${GO_VERSION:-}"
INSTALL_QUICKSHELL="${INSTALL_QUICKSHELL:-1}"
PROVISION_DEBUG="${PROVISION_DEBUG:-0}"

LOG=/var/log/i3-next-provision.log
exec > >(tee -a "$LOG") 2>&1

START=$SECONDS
STEP_N=0
STEP_START=$SECONDS
STEP_NAME=""

ts()   { date '+%H:%M:%S'; }
info() { printf '%s \033[1;34m[guest]\033[0m %s\n' "$(ts)" "$*"; }
warn() { printf '%s \033[1;33m[guest warn]\033[0m %s\n' "$(ts)" "$*"; }
step() {
  if [ -n "$STEP_NAME" ]; then info "<- done: $STEP_NAME ($((SECONDS - STEP_START))s)"; fi
  STEP_N=$((STEP_N + 1)); STEP_NAME="$*"; STEP_START=$SECONDS
  printf '\n%s \033[1;36m[guest] === step %d: %s ===\033[0m\n' "$(ts)" "$STEP_N" "$STEP_NAME"
}

on_err() {
  local line="$1" cmd="$2" code="$3"
  printf '\n%s \033[1;31m[guest ERROR]\033[0m step %d "%s" failed\n' "$(ts)" "$STEP_N" "$STEP_NAME"
  printf '  line:    %s\n  command: %s\n  exit:    %s\n  log:     %s\n' "$line" "$cmd" "$code" "$LOG"
}
on_exit() {
  local code=$?
  printf '%s [guest] finished with exit code %d after %ds\n' "$(ts)" "$code" "$((SECONDS - START))"
}
trap 'on_err "$LINENO" "$BASH_COMMAND" "$?"' ERR
trap on_exit EXIT

if [ "$PROVISION_DEBUG" = 1 ]; then
  export PS4='+ $(date +%T) ${BASH_SOURCE##*/}:${LINENO}: '
  set -x
fi

REQUIRED=(
  git curl ca-certificates build-essential pkg-config cmake ninja-build make
  jq rsync file dbus dbus-user-session libpam-systemd
  sway swaybg swayidle swaylock waybar wl-clipboard grim slurp foot
  pipewire wireplumber libwayland-dev wayland-protocols libgl1-mesa-dri
  qemu-guest-agent fonts-dejavu-core
)
OPTIONAL=(wtype wayvnc xdg-desktop-portal-wlr spice-vdagent)

# ---------------------------------------------------------------------------
step "preflight: system and network diagnostics"
info "provision log: $LOG   (debug tracing: PROVISION_DEBUG=$PROVISION_DEBUG)"
info "os:     $(. /etc/os-release && echo "$PRETTY_NAME")  kernel $(uname -r)  arch $(uname -m)"
info "cpu:    $(nproc) vCPU, mem: $(free -m | awk '/Mem:/ {print $2" MiB total, "$7" MiB available"}')"
info "disk:"; df -h / | sed 's/^/        /'
info "routes:"; ip -brief route 2>/dev/null | sed 's/^/        /' || true
info "addrs:";  ip -brief addr  2>/dev/null | sed 's/^/        /' || true
info "dns:      $(grep -E '^(nameserver|search)' /etc/resolv.conf 2>/dev/null | tr '\n' ' ')"
info "apt sources:"
{ cat /etc/apt/sources.list 2>/dev/null; cat /etc/apt/sources.list.d/* 2>/dev/null; } |
  grep -vE '^\s*(#|$)' | sed 's/^/        /' || true

net_ok=1
for host in deb.debian.org go.dev; do
  if getent hosts "$host" >/dev/null; then
    code="$(curl -sS -o /dev/null -m 15 -w '%{http_code}' "https://$host/" 2>&1 || true)"
    info "reach $host: DNS ok, HTTPS status $code"
    case "$code" in 2*|3*|4*) ;; *) net_ok=0 ;; esac
  else
    warn "cannot resolve $host"; net_ok=0
  fi
done
if [ "$net_ok" != 1 ]; then
  warn "network problems detected; apt/Go downloads will probably fail."
  warn "check on the host: virsh net-list --all ; virsh net-info default ; firewall (nft/iptables) forwarding rules"
fi

# ---------------------------------------------------------------------------
step "apt: refresh package index"
apt-get update -o Dpkg::Progress-Fancy=0

step "apt: check package availability"
missing=()
for p in "${REQUIRED[@]}"; do
  if apt-cache show "$p" >/dev/null 2>&1; then
    info "available: $p ($(apt-cache policy "$p" | awk '/Candidate:/ {print $2}'))"
  else
    warn "NOT AVAILABLE: $p"; missing+=("$p")
  fi
done
if [ "${#missing[@]}" -gt 0 ]; then
  info "required packages missing from the configured repositories: ${missing[*]}"
  exit 1
fi

step "apt: install required packages (${#REQUIRED[@]})"
apt-get install -y -o Dpkg::Progress-Fancy=0 "${REQUIRED[@]}"

step "apt: install optional packages (failures are non-fatal)"
for p in "${OPTIONAL[@]}"; do
  if apt-get install -y -o Dpkg::Progress-Fancy=0 "$p"; then info "installed optional: $p"
  else warn "optional package skipped: $p"; fi
done

# ---------------------------------------------------------------------------
step "go: install toolchain"
install_go() {
  if [ -z "$GO_VERSION" ]; then
    warn "go.mod version unknown, installing distro golang"
    apt-get install -y golang-go
    return
  fi
  local v="$GO_VERSION" arch url code
  if [[ "$v" =~ ^[0-9]+\.[0-9]+$ ]]; then v="$v.0"; fi
  info "wanted Go version: $v (from go.mod: $GO_VERSION)"
  if [ -x /usr/local/go/bin/go ] && /usr/local/go/bin/go version | grep -q "go$v "; then
    info "Go $v already installed: $(/usr/local/go/bin/go version)"
    return
  fi
  case "$(uname -m)" in
    x86_64)  arch=amd64 ;;
    aarch64) arch=arm64 ;;
    *) warn "unsupported arch $(uname -m) for the Go download"; return 1 ;;
  esac
  url="https://go.dev/dl/go${v}.linux-${arch}.tar.gz"
  code="$(curl -sS -o /dev/null -I -L -m 20 -w '%{http_code}' "$url" || true)"
  info "HEAD $url -> HTTP $code"
  if [ "$code" != 200 ]; then
    warn "Go $v is not downloadable (HTTP $code). Does this release exist? Check the 'go' line in go.mod,"
    warn "or list releases at https://go.dev/dl/ ."
    return 1
  fi
  curl -fSL --retry 3 -w 'download: http=%{http_code} size=%{size_download}B time=%{time_total}s\n' \
    -o /tmp/go.tgz "$url"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf /tmp/go.tgz
  rm -f /tmp/go.tgz
  ln -sf /usr/local/go/bin/go /usr/local/bin/go
  ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
  info "installed: $(/usr/local/go/bin/go version)"
}
install_go

step "quickshell: install (optional)"
if [ "$INSTALL_QUICKSHELL" = 1 ]; then
  if apt-cache show quickshell >/dev/null 2>&1; then
    apt-get install -y -o Dpkg::Progress-Fancy=0 quickshell
  else
    warn "quickshell is not in this distro's apt repositories; install it manually or build from source"
  fi
else
  info "skipped (INSTALL_QUICKSHELL=0)"
fi

# ---------------------------------------------------------------------------
step "session: environment, autologin and profile hook"
cat > /etc/i3-next-test.env <<'ENV'
# Software rendering is the most robust choice on a virtual GPU.
# Remove WLR_RENDERER when running with VM_GL=1 on the host.
WLR_RENDERER=pixman
WLR_NO_HARDWARE_CURSORS=1
XDG_SESSION_TYPE=wayland
XDG_CURRENT_DESKTOP=sway
ENV
info "wrote /etc/i3-next-test.env"

mkdir -p /etc/systemd/system/getty@tty1.service.d
cat > /etc/systemd/system/getty@tty1.service.d/autologin.conf <<UNIT
[Service]
ExecStart=
ExecStart=-/sbin/agetty --autologin ${DEV_USER} --noclear %I \$TERM
UNIT
systemctl daemon-reload
info "wrote getty@tty1 autologin drop-in for user '$DEV_USER'"

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
  info "appended session hook to $PROFILE"
else
  info "session hook already present in $PROFILE"
fi

loginctl enable-linger "$DEV_USER"
info "linger enabled for $DEV_USER"
systemctl start qemu-guest-agent 2>/dev/null || warn "qemu-guest-agent did not start (non-fatal)"

# ---------------------------------------------------------------------------
step "verify: required tools and versions"
fail=0
for t in sway swaymsg grim slurp foot jq rsync git make gcc go; do
  if path="$(command -v "$t")"; then info "found $t -> $path"
  else warn "MISSING tool: $t"; fail=1; fi
done
info "versions:"
{
  sway --version 2>&1 | head -n1
  go version 2>&1
  foot --version 2>&1 | head -n1
  jq --version 2>&1
  rsync --version 2>&1 | head -n1
  quickshell --version 2>&1 | head -n1 || echo "quickshell: not installed"
} | sed 's/^/        /'
[ "$fail" = 0 ] || { warn "some required tools are missing, see above"; exit 1; }

mkdir -p /var/lib/i3-next-vm
date -u +%FT%TZ > /var/lib/i3-next-vm/provisioned
step "complete"
info "provisioning OK (marker: /var/lib/i3-next-vm/provisioned)"