#!/usr/bin/env python3
"""Idempotent Debian guest provisioning for the i3-next KVM lab."""
from __future__ import annotations

import datetime as dt
import os
import platform
import shutil
import subprocess
import sys
import time
import urllib.request
from pathlib import Path
import re

LOG = Path("/var/log/i3-next-provision.log")
DEV_USER = os.getenv("DEV_USER", "dev")
GO_VERSION = os.getenv("GO_VERSION", "")
INSTALL_QUICKSHELL = os.getenv("INSTALL_QUICKSHELL", "1") == "1"
DEBUG = os.getenv("PROVISION_DEBUG", "0") == "1"
REQUIRED = [
    "curl", "ca-certificates", "build-essential", "pkg-config", "cmake",
    "ninja-build", "make", "jq", "rsync", "file", "dbus", "dbus-user-session",
    "libpam-systemd", "sway", "swaybg", "swayidle", "swaylock", "waybar",
    "wl-clipboard", "grim", "slurp", "foot", "pipewire", "wireplumber",
    "libwayland-dev", "wayland-protocols", "libgl1-mesa-dri",
    "qemu-guest-agent", "fonts-dejavu-core",
]
OPTIONAL = ["wtype", "wayvnc", "xdg-desktop-portal-wlr", "spice-vdagent"]
START = time.monotonic()
STEP = 0
STEP_NAME = ""
STEP_START = START

def command(args, *, check=True, capture=False, timeout=None):
    args = [str(x) for x in args]
    if DEBUG:
        info("$ " + " ".join(args))
    p = subprocess.run(args, text=True, stdout=subprocess.PIPE if capture else None,
                       stderr=subprocess.STDOUT if capture else None, timeout=timeout)
    if check and p.returncode:
        raise RuntimeError(f"command failed ({p.returncode}): {' '.join(args)}")
    return p.stdout.strip() if capture else p.returncode

def ts():
    return dt.datetime.now().astimezone().strftime("%H:%M:%S")

def info(message):
    print(f"{ts()} \033[1;34m[guest]\033[0m {message}", flush=True)

def warn(message):
    print(f"{ts()} \033[1;33m[guest warn]\033[0m {message}", flush=True)

def step(name):
    global STEP, STEP_NAME, STEP_START
    if STEP_NAME:
        info(f"<- done: {STEP_NAME} ({int(time.monotonic()-STEP_START)}s)")
    STEP += 1
    STEP_NAME = name
    STEP_START = time.monotonic()
    print(f"\n{ts()} \033[1;36m[guest] === step {STEP}: {name} ===\033[0m", flush=True)

def append_file(path: Path, content: str, mode=0o644):
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("a") as stream:
        stream.write(content)
    path.chmod(mode)

def check_root():
    if os.geteuid() != 0:
        raise RuntimeError("guest-provision.py must run as root")

def preflight():
    step("preflight: system and network diagnostics")
    info(f"provision log: {LOG} (debug tracing: PROVISION_DEBUG={int(DEBUG)})")
    try:
        os_release = Path("/etc/os-release").read_text()
    except OSError:
        os_release = "unknown"
    info(f"os: {next((x.split('=',1)[1].strip(chr(34)) for x in os_release.splitlines() if x.startswith('PRETTY_NAME=')), 'unknown')} kernel {platform.release()} arch {platform.machine()}")
    info(f"cpu: {os.cpu_count()} vCPU")
    for label, args in (("disk", ["df", "-h", "/"]), ("routes", ["ip", "-brief", "route"]),
                        ("addrs", ["ip", "-brief", "addr"])):
        output = command(args, check=False, capture=True)
        info(f"{label}:\n        " + output.replace("\n", "\n        "))
    dns = Path("/etc/resolv.conf").read_text() if Path("/etc/resolv.conf").exists() else ""
    info("dns: " + " ".join(x for x in dns.splitlines() if x.startswith(("nameserver", "search"))))
    net_ok = True
    for host in ("deb.debian.org", "go.dev"):
        resolved = command(["getent", "hosts", host], check=False, capture=True)
        if resolved:
            try:
                req = urllib.request.Request(f"https://{host}/", method="HEAD")
                with urllib.request.urlopen(req, timeout=15) as response:
                    code = response.status
            except Exception as exc:
                code = str(exc)
            info(f"reach {host}: DNS ok, HTTPS status {code}")
            if isinstance(code, int) and not (200 <= code < 500):
                net_ok = False
            elif not isinstance(code, int):
                net_ok = False
        else:
            warn(f"cannot resolve {host}")
            net_ok = False
    if not net_ok:
        warn("network problems detected; apt/Go downloads will probably fail.")
        warn("check host libvirt network, forwarding and firewall rules")

def install_packages():
    step("apt: refresh package index")
    # Configure Debian repositories first.
    configure_debian_repositories()
    # Refresh package indexes.
    command(["apt-get", "update", "-o", "Dpkg::Progress-Fancy=0"], check=False)
    step("apt: check package availability")
    # Install required packages.
    install_required_packages()


def install_required_packages():
    missing = []
    for package in REQUIRED:
        if command(["apt-cache", "show", package], check=False, capture=True):
            candidate = command(["apt-cache", "policy", package], check=False, capture=True)
            candidate = next((line.split(":",1)[1].strip() for line in candidate.splitlines()
                              if "Candidate:" in line), "unknown")
            info(f"available: {package} ({candidate})")
        else:
            warn(f"NOT AVAILABLE: {package}")
            missing.append(package)
    if missing:
        raise RuntimeError("required packages missing from configured repositories: " + " ".join(missing))
    step(f"apt: install required packages ({len(REQUIRED)})")
    command(["apt-get", "install", "-y", "-o", "Dpkg::Progress-Fancy=0", *REQUIRED])
    step("apt: install optional packages (failures are non-fatal)")
    for package in OPTIONAL:
        if command(["apt-get", "install", "-y", "-o", "Dpkg::Progress-Fancy=0", package], check=False) == 0:
            info(f"installed optional: {package}")
        else:
            warn(f"optional package skipped: {package}")

def parse_go_version(text: str) -> tuple[int, ...]:
    match = re.search(r"(\d+)\.(\d+)(?:\.(\d+))?", text)
    if not match:
        return ()
    return tuple(int(x) for x in match.groups(default="0"))

def installed_go_version() -> tuple[int, ...]:
    go = shutil.which("go")
    if not go:
        return ()
    return parse_go_version(command([go, "version"], check=False, capture=True))

def install_go():
    step("go: install toolchain (from apt)")
    required = parse_go_version(GO_VERSION) if GO_VERSION else ()
    if required:
        info(f"required Go version (from go.mod): {'.'.join(map(str, required))}")
    else:
        warn("go.mod version unknown; any Go from apt is accepted")

    # Remove leftovers of the old tarball install: /usr/local/bin precedes
    # /usr/bin in PATH, so a stale symlink would shadow the apt package.
    for binary in ("go", "gofmt"):
        link = Path("/usr/local/bin") / binary
        if link.is_symlink() and str(link.resolve()).startswith("/usr/local/go"):
            info(f"removing stale symlink {link}")
            link.unlink()
    if Path("/usr/local/go").exists():
        info("removing old /usr/local/go")
        shutil.rmtree("/usr/local/go", ignore_errors=True)

    current = installed_go_version()
    if current and current >= required:
        info(f"Go already installed: {'.'.join(map(str, current))}")
    else:
        command(["apt-get", "install", "-y", "-o", "Dpkg::Progress-Fancy=0", "golang-go"])
        current = installed_go_version()
        if not current:
            raise RuntimeError("golang-go installed but 'go' was not found in PATH")
        if current < required:
            raise RuntimeError(
                f"apt provides Go {'.'.join(map(str, current))}, but go.mod needs "
                f"{'.'.join(map(str, required))}. Lower the 'go' line in go.mod "
                "or enable trixie-backports.")
        info("installed: " + command(["go", "version"], capture=True))

    # Never let `go` silently download a newer toolchain over the network.
    command(["runuser", "-u", DEV_USER, "--", "env", f"HOME=/home/{DEV_USER}",
             "go", "env", "-w", "GOTOOLCHAIN=local"])
    info(f"GOTOOLCHAIN=local set for {DEV_USER}")

def install_quickshell():
    step("quickshell: install (optional)")
    if not INSTALL_QUICKSHELL:
        info("skipped (INSTALL_QUICKSHELL=0)")
        return
    if not command(["apt-cache", "show", "quickshell"], check=False, capture=True):
        warn("quickshell is not in apt repositories; install manually or build from source")
        return
    rc = command(["apt-get", "install", "-y", "-o", "Dpkg::Progress-Fancy=0", "quickshell"],
                 check=False)
    if rc != 0:
        warn(f"quickshell install failed (apt exit {rc}); continuing without it")
        warn("see: apt-get install -s quickshell  (usually unmet Qt6 dependencies)")
        return
    info("installed: " + command(["quickshell", "--version"], check=False, capture=True))

def configure_session():
    step("session: environment, autologin and profile hook")
    Path("/etc/i3-next-test.env").write_text(
        "# Software rendering is the most robust choice on a virtual GPU.\n"
        "# Remove WLR_RENDERER when running with VM_GL=1 on the host.\n"
        "WLR_RENDERER=pixman\nWLR_NO_HARDWARE_CURSORS=1\n"
        "XDG_SESSION_TYPE=wayland\nXDG_CURRENT_DESKTOP=sway\n")
    dropin = Path("/etc/systemd/system/getty@tty1.service.d/autologin.conf")
    dropin.parent.mkdir(parents=True, exist_ok=True)
    dropin.write_text("[Service]\nExecStart=\nExecStart=-/sbin/agetty --autologin "
                      f"{DEV_USER} --noclear %I $TERM\n")
    command(["systemctl", "daemon-reload"])
    profile = Path(f"/home/{DEV_USER}/.profile")
    marker = ">>> i3-next-test >>>"
    if marker not in (profile.read_text() if profile.exists() else ""):
        append_file(profile, """
# >>> i3-next-test >>>
# Start Sway on tty1 only when vm.py gui on enabled the flag.
if [ -z "${WAYLAND_DISPLAY:-}" ] && [ "$(tty 2>/dev/null)" = /dev/tty1 ] && [ -f /etc/i3-next-gui-session ]; then
  set -a; . /etc/i3-next-test.env; set +a
  cfg="$HOME/i3-next/configs/sway/config"
  if [ -s "$cfg" ]; then exec sway -c "$cfg"; else exec sway; fi
fi
# <<< i3-next-test <<<
""")
        info(f"appended session hook to {profile}")
    else:
        info(f"session hook already present in {profile}")
    command(["loginctl", "enable-linger", DEV_USER])
    info(f"linger enabled for {DEV_USER}")
    if command(["systemctl", "start", "qemu-guest-agent"], check=False) != 0:
        warn("qemu-guest-agent did not start (non-fatal)")

def verify():
    step("verify: required tools and versions")
    required = ("sway", "swaymsg", "grim", "slurp", "foot", "jq", "rsync", "make", "gcc", "go")
    missing = []
    for tool in required:
        path = shutil.which(tool)
        if path:
            info(f"found {tool} -> {path}")
        else:
            warn(f"MISSING tool: {tool}")
            missing.append(tool)
    for tool, args in (("sway", ["sway", "--version"]), ("go", ["go", "version"]),
                       ("foot", ["foot", "--version"]), ("jq", ["jq", "--version"]),
                       ("rsync", ["rsync", "--version"]), ("quickshell", ["quickshell", "--version"])):
        result = command(args, check=False, capture=True)
        info(f"{tool}: {result.splitlines()[0] if result else 'not installed'}")
    if missing:
        raise RuntimeError("required tools missing: " + ", ".join(missing))
    marker = Path("/var/lib/i3-next-vm/provisioned")
    marker.parent.mkdir(parents=True, exist_ok=True)
    marker.write_text(dt.datetime.now(dt.timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ") + "\n")

def configure_debian_repositories():
    """Ensure Debian 13 (Trixie) repositories are configured."""

    os_release = {}

    for line in Path("/etc/os-release").read_text().splitlines():
        if "=" in line:
            key, value = line.split("=", 1)
            os_release[key] = value.strip().strip('"')

    if os_release.get("ID") != "debian":
        print("Skipping Debian repository configuration: not Debian")
        return

    codename = os_release.get("VERSION_CODENAME")

    if codename != "trixie":
        print(f"Skipping repository configuration for Debian {codename}")
        return

    sources = """\
Types: deb
URIs: https://deb.debian.org/debian
Suites: trixie trixie-updates trixie-backports
Components: main contrib non-free non-free-firmware
Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg

Types: deb
URIs: https://security.debian.org/debian-security
Suites: trixie-security
Components: main contrib non-free non-free-firmware
Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg
"""

    sources_file = Path(
        "/etc/apt/sources.list.d/debian.sources"
    )

    sources_file.write_text(sources)
    sources_file.chmod(0o644)

    print(f"Configured Debian repositories: {sources_file}")

def main():
    check_root()
    os.environ["DEBIAN_FRONTEND"] = "noninteractive"
    LOG.parent.mkdir(parents=True, exist_ok=True)
    with LOG.open("a", buffering=1) as log_file:
        original = sys.stdout
        class Tee:
            def write(self, value):
                original.write(value); log_file.write(value)
            def flush(self):
                original.flush(); log_file.flush()
        sys.stdout = Tee()
        try:
            preflight()
            install_packages()
            install_go()
            install_quickshell()
            configure_session()
            verify()
            step("complete")
            info("provisioning OK (marker: /var/lib/i3-next-vm/provisioned)")
        except Exception as exc:
            print(f"\n{ts()} [guest ERROR] step {STEP} {STEP_NAME!r} failed: {exc}", flush=True)
            print(f"  log: {LOG}", flush=True)
            raise
        finally:
            info(f"finished after {int(time.monotonic()-START)}s")
            sys.stdout = original

if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"[guest fatal] {exc}", file=sys.stderr)
        raise SystemExit(1)
