#!/usr/bin/env python3
"""Python implementation of the i3-next KVM/libvirt test-lab manager."""
from __future__ import annotations

import argparse
import datetime as dt
import os
import re
import shlex
import shutil
import subprocess
import sys
import time
import urllib.request
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = SCRIPT_DIR.parents[1]
ENV = os.environ

def setting(name: str, default: str) -> str:
    return ENV.get(name, default)

VM_NAME = setting("VM_NAME", "i3-next-test")
VM_USER = setting("VM_USER", "dev")
VM_CPUS = setting("VM_CPUS", "4")
VM_RAM_MB = setting("VM_RAM_MB", "4096")
VM_DISK_GB = setting("VM_DISK_GB", "30")
VM_GRAPHICS = setting("VM_GRAPHICS", "spice")
VM_GL = setting("VM_GL", "0")
VM_NETWORK = setting("VM_NETWORK", "default")
VM_POOL = setting("VM_POOL", "default")
LIBVIRT_URI = setting("LIBVIRT_URI", "qemu:///system")
IMAGE_BASE_URL = setting("IMAGE_BASE_URL", "https://cloud.debian.org/images/cloud/trixie/latest")
IMAGE_FILE = setting("IMAGE_FILE", "debian-13-generic-amd64.qcow2")
SNAPSHOT_NAME = setting("SNAPSHOT_NAME", "provisioned")
STATE_DIR = Path(setting("VM_STATE_DIR", str(SCRIPT_DIR / ".state"))).expanduser()
ARTIFACTS_ROOT = STATE_DIR / "artifacts"
KEY = STATE_DIR / "id_ed25519"
BASE_VOL = f"i3-next-base-{IMAGE_FILE}"
DISK_VOL = f"{VM_NAME}.qcow2"
SEED_VOL = f"{VM_NAME}-seed.iso"
SSH_BASE = ["-i", str(KEY), "-o", "IdentitiesOnly=yes",
            "-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile=/dev/null",
            "-o", "LogLevel=ERROR", "-o", "ConnectTimeout=5",
            "-o", "ServerAliveInterval=15"]

class VMError(RuntimeError):
    pass

def log(message: str) -> None:
    print(f"\033[1;34m==>\033[0m {message}", file=sys.stderr)

def warn(message: str) -> None:
    print(f"\033[1;33m[warn]\033[0m {message}", file=sys.stderr)

def run(args, *, check=True, capture=False, input_text=None, env=None):
    args = [str(x) for x in args]
    if capture:
        p = subprocess.run(args, text=True, stdout=subprocess.PIPE,
                           stderr=subprocess.PIPE, input=input_text, env=env)
        if check and p.returncode:
            raise VMError(f"{shlex.join(args)} failed ({p.returncode}): {p.stderr.strip()}")
        return p.stdout.strip()
    p = subprocess.run(args, text=True, input=input_text, env=env)
    if check and p.returncode:
        raise VMError(f"{shlex.join(args)} failed ({p.returncode})")
    return p.returncode

def virsh(*args, check=True, capture=True):
    return run(["virsh", "-c", LIBVIRT_URI, *args], check=check, capture=capture)

def have(command: str) -> bool:
    return shutil.which(command) is not None

def vm_exists() -> bool:
    return virsh("dominfo", VM_NAME, check=False) != ""

def vm_state() -> str:
    return virsh("domstate", VM_NAME, check=False).splitlines()[0].strip() if virsh("domstate", VM_NAME, check=False) else ""

def vm_running() -> bool:
    return vm_state() == "running"

def vm_ip() -> str:
    for source in ("lease", "agent", "arp"):
        output = virsh("domifaddr", VM_NAME, "--source", source, check=False)
        for line in output.splitlines():
            cols = line.split()
            if len(cols) >= 4 and "ipv4" in cols and not cols[3].startswith("127."):
                return cols[3].split("/")[0]
    raise VMError(f"VM {VM_NAME!r} has no IP address (is it running?)")

def ssh_args(command=None):
    ip = vm_ip()
    args = ["ssh", *SSH_BASE, f"{VM_USER}@{ip}"]
    if command:
        args.append(command if isinstance(command, str) else shlex.join(command))
    return args

def guest(command, *, check=True, capture=False, input_text=None):
    return run(ssh_args(command), check=check, capture=capture, input_text=input_text)

def rsync_shell() -> str:
    return " ".join(shlex.quote(x) for x in ["ssh", *SSH_BASE])

def ensure_key():
    STATE_DIR.mkdir(parents=True, exist_ok=True)
    if not KEY.exists():
        run(["ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-C",
             "i3-next-test", "-f", KEY])

def ensure_running():
    if not vm_exists():
        raise VMError(f"VM {VM_NAME!r} does not exist. Run: vm.py up")
    if not vm_running():
        cmd_start()

def wait_for_ssh(timeout=240):
    log(f"waiting for SSH (up to {timeout}s)")
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            ip = vm_ip()
            p = subprocess.run(["ssh", *SSH_BASE, "-o", "BatchMode=yes",
                                f"{VM_USER}@{ip}", "true"],
                               stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            if p.returncode == 0:
                log(f"SSH is up on {ip}")
                return
        except VMError:
            pass
        time.sleep(3)
    raise VMError("timed out waiting for SSH")

def pool_vol_exists(name):
    return virsh("vol-info", "--pool", VM_POOL, name, check=False) != ""

def pool_vol_delete(name):
    if pool_vol_exists(name):
        virsh("vol-delete", "--pool", VM_POOL, name)

def pool_vol_upload(name: str, file: Path, fmt: str):
    size = file.stat().st_size
    virsh("vol-create-as", VM_POOL, name, f"{size}B", "--format", fmt)
    virsh("vol-upload", "--pool", VM_POOL, name, str(file), capture=False)

def ensure_provisioned():
    if guest("test -f /var/lib/i3-next-vm/provisioned", check=False) != 0:
        warn("guest is not provisioned; provisioning now")
        cmd_provision(None)

def cmd_doctor(_args):
    checks = []
    checks.append(("CPU virtualization (vmx/svm)", lambda: re.search(r"\b(vmx|svm)\b", Path("/proc/cpuinfo").read_text()) is not None))
    checks.append(("/dev/kvm exists and is accessible", lambda: os.access("/dev/kvm", os.R_OK | os.W_OK)))
    for tool in ("virsh", "virt-install", "qemu-img", "ssh", "ssh-keygen", "rsync", "curl", "sha512sum"):
        checks.append((f"tool: {tool}", lambda t=tool: have(t)))
    checks.append(("ISO tool (cloud-localds|genisoimage|xorriso|mkisofs)",
                   lambda: any(have(x) for x in ("cloud-localds", "genisoimage", "xorriso", "mkisofs"))))
    checks.append((f"libvirt reachable at {LIBVIRT_URI}", lambda: virsh("list", check=False) is not None))
    checks.append((f"storage pool {VM_POOL!r}", lambda: virsh("pool-info", VM_POOL, check=False) != ""))
    checks.append((f"network {VM_NETWORK!r} active", lambda: "Active: yes" in virsh("net-info", VM_NETWORK, check=False)))
    failed = False
    print("Host checks:")
    for label, fn in checks:
        try:
            ok = bool(fn())
        except Exception:
            ok = False
        print(f"  [{'OK' if ok else 'FAIL'}] {label}")
        failed |= not ok
    if not have("virt-viewer"):
        warn("virt-viewer not found: viewer/gui unavailable (headless tests are fine)")
    if failed:
        print("""\nHints (Debian/Ubuntu):
  sudo apt install qemu-system-x86 libvirt-daemon-system libvirt-clients virtinst \\
                   virt-viewer genisoimage rsync curl
  sudo usermod -aG libvirt,kvm "$USER"   # then log out/in
  sudo virsh net-start default; sudo virsh net-autostart default
  sudo virsh pool-define-as default dir --target /var/lib/libvirt/images
  sudo virsh pool-start default; sudo virsh pool-autostart default""", file=sys.stderr)
        raise VMError("host prerequisite checks failed")

def cmd_image(_args):
    base_dir = STATE_DIR / "base"
    base_dir.mkdir(parents=True, exist_ok=True)
    dst = base_dir / IMAGE_FILE
    if not dst.exists():
        log(f"downloading {IMAGE_FILE}")
        tmp = dst.with_suffix(dst.suffix + ".part")
        urllib.request.urlretrieve(f"{IMAGE_BASE_URL}/{IMAGE_FILE}", tmp)
        tmp.replace(dst)
    log("checking Debian image SHA512")
    sums = urllib.request.urlopen(f"{IMAGE_BASE_URL}/SHA512SUMS", timeout=30).read().decode()
    expected = next((line.split()[0] for line in sums.splitlines()
                     if len(line.split()) >= 2 and line.split()[-1].lstrip("*") == IMAGE_FILE), None)
    if not expected:
        raise VMError(f"no checksum published for {IMAGE_FILE}")
    import hashlib
    digest = hashlib.sha512(dst.read_bytes()).hexdigest()
    if digest != expected:
        # dst.unlink(missing_ok=True)
        raise VMError("Debian cloud image SHA512 mismatch")
    log("base image SHA512 verified")

def make_seed_iso():
    seed_dir = STATE_DIR / "seed"
    seed_dir.mkdir(parents=True, exist_ok=True)
    pub = Path(f"{KEY}.pub").read_text().strip()
    user_data = (SCRIPT_DIR / "cloud-init/user-data.tpl").read_text()
    meta_data = (SCRIPT_DIR / "cloud-init/meta-data.tpl").read_text()
    user_data = (user_data.replace("__HOSTNAME__", VM_NAME).replace("__USER__", VM_USER)
                 .replace("__SSH_PUBKEY__", pub))
    meta_data = meta_data.replace("__HOSTNAME__", VM_NAME).replace(
        "__INSTANCE__", str(int(time.time())))
    (seed_dir / "user-data").write_text(user_data)
    (seed_dir / "meta-data").write_text(meta_data)
    output = STATE_DIR / "seed.iso"
    if have("cloud-localds"):
        run(["cloud-localds", output, seed_dir / "user-data", seed_dir / "meta-data"])
    elif have("genisoimage"):
        run(["genisoimage", "-quiet", "-output", output, "-volid", "cidata",
             "-joliet", "-rock", seed_dir / "user-data", seed_dir / "meta-data"])
    elif have("xorriso"):
        run(["xorriso", "-as", "mkisofs", "-quiet", "-o", output, "-V",
             "cidata", "-J", "-r", seed_dir / "user-data", seed_dir / "meta-data"])
    elif have("mkisofs"):
        run(["mkisofs", "-quiet", "-output", output, "-volid", "cidata",
             "-joliet", "-rock", seed_dir / "user-data", seed_dir / "meta-data"])
    else:
        raise VMError("need one of: cloud-localds, genisoimage, xorriso, mkisofs")
    return output

def pick_os_variant():
    if have("osinfo-query"):
        for value in ("debian13", "debian12", "linux2022", "generic"):
            if run(["osinfo-query", "os", f"short-id={value}"], check=False) == 0:
                return value
    return "generic"

def cmd_create(_args):
    if vm_exists():
        raise VMError(f"VM {VM_NAME!r} already exists (use: vm.py recreate)")
    ensure_key()
    cmd_image(None)
    base_file = STATE_DIR / "base" / IMAGE_FILE
    if not pool_vol_exists(BASE_VOL):
        log(f"uploading base image into pool {VM_POOL!r}")
        pool_vol_upload(BASE_VOL, base_file, "qcow2")
    pool_vol_delete(DISK_VOL)
    virsh("vol-create-as", VM_POOL, DISK_VOL, f"{VM_DISK_GB}G",
          "--format", "qcow2", "--backing-vol", BASE_VOL, "--backing-vol-format", "qcow2")
    seed = make_seed_iso()
    pool_vol_delete(SEED_VOL)
    pool_vol_upload(SEED_VOL, seed, "raw")
    if VM_GRAPHICS == "spice":
        gfx = (["--graphics", "spice,gl.enable=yes,listen=none",
                "--video", "model.type=virtio,model.acceleration.accel3d=yes"] if VM_GL == "1"
               else ["--graphics", "spice,listen=none", "--video", "virtio"])
        gfx += ["--channel", "spicevmc"]
    elif VM_GRAPHICS == "vnc":
        gfx = ["--graphics", "vnc,listen=127.0.0.1", "--video", "virtio"]
    elif VM_GRAPHICS == "none":
        gfx = ["--graphics", "none"]
    else:
        raise VMError("VM_GRAPHICS must be spice, vnc or none")
    log(f"defining and starting {VM_NAME!r} ({VM_CPUS} vCPU, {VM_RAM_MB} MiB, graphics={VM_GRAPHICS})")
    run(["virt-install", "--connect", LIBVIRT_URI, "--name", VM_NAME,
         "--memory", VM_RAM_MB, "--vcpus", VM_CPUS, "--cpu", "host-passthrough",
         "--os-variant", pick_os_variant(), "--import", "--boot", "hd",
         "--disk", f"vol={VM_POOL}/{DISK_VOL},bus=virtio",
         "--disk", f"vol={VM_POOL}/{SEED_VOL},device=cdrom,bus=sata",
         "--network", f"network={VM_NETWORK},model=virtio", *gfx,
         "--channel", "unix,target_type=virtio,name=org.qemu.guest_agent.0",
         "--rng", "/dev/urandom", "--noautoconsole"])
    wait_for_ssh(300)
    cmd_provision(None)
    target = ""
    for line in virsh("domblklist", VM_NAME, "--details").splitlines():
        cols = line.split()
        if len(cols) >= 3 and cols[1] == "cdrom":
            target = cols[2]
            break
    if target:
        if virsh("change-media", VM_NAME, target, "--eject", "--force",
                 "--live", "--config", check=False) == "":
            warn("could not eject cloud-init seed")
    # IMPORTANT: baseline snapshot is created after Go/Wayland/Sway provisioning,
    # and before any repository sync/build/install.
    cmd_snapshot(argparse.Namespace(name=SNAPSHOT_NAME))
    log("lab is ready. Try: vm.py test | vm.py gui | vm.py ssh")

def cmd_provision(_args):
    ensure_running()
    go_version = ""
    go_mod = REPO_ROOT / "go.mod"
    if go_mod.exists():
        match = re.search(r"(?m)^go\s+([0-9.]+)", go_mod.read_text())
        go_version = match.group(1) if match else ""
    log_dir = STATE_DIR / "logs"
    log_dir.mkdir(parents=True, exist_ok=True)
    logfile = log_dir / f"provision-{dt.datetime.now():%Y%m%d-%H%M%S}.log"
    log(f"provisioning guest (Go {go_version or 'distro'}); host log: {logfile}")
    guest("sudo cloud-init status --wait --long || true")
    provision_src = (SCRIPT_DIR / "guest-provision.py").read_text()
    guest("cat > /tmp/guest-provision.py", input_text=provision_src)
    env = {
        "GO_VERSION": go_version, "DEV_USER": VM_USER,
        "INSTALL_QUICKSHELL": setting("INSTALL_QUICKSHELL", "1"),
        "PROVISION_DEBUG": setting("PROVISION_DEBUG", "0"),
    }
    remote_env = " ".join(f"{k}={shlex.quote(v)}" for k, v in env.items())
    p = subprocess.run(ssh_args(f"sudo env {remote_env} python3 /tmp/guest-provision.py"),
                       text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    logfile.write_text(p.stdout)
    print(p.stdout, end="")
    if p.returncode:
        warn(f"provisioning FAILED (exit {p.returncode})")
        warn(f"host log: {logfile}")
        warn("guest log: /var/log/i3-next-provision.log (vm.py logs)")
        try:
            tail = guest("sudo tail -n 25 /var/log/i3-next-provision.log", capture=True)
            print("\n".join("    | " + line for line in tail.splitlines()), file=sys.stderr)
        except Exception:
            pass
        raise VMError("guest provisioning failed")

def cmd_logs(_args):
    ensure_running()
    guest("sudo cat /var/log/i3-next-provision.log")

def cmd_up(_args):
    if not vm_exists():
        cmd_create(None)
    elif not vm_running():
        cmd_start(None)
    else:
        wait_for_ssh(60)
    ensure_provisioned()

def cmd_start(_args):
    if not vm_exists():
        raise VMError(f"VM {VM_NAME!r} does not exist. Run: vm.py up")
    if not vm_running():
        virsh("start", VM_NAME)
    wait_for_ssh(240)

def cmd_stop(_args):
    if not vm_running():
        log("VM already stopped")
        return
    virsh("shutdown", VM_NAME)
    for _ in range(30):
        if not vm_running():
            log("VM stopped")
            return
        time.sleep(2)
    warn("graceful shutdown timed out, forcing power off")
    virsh("destroy", VM_NAME)

def cmd_destroy(args):
    if vm_exists():
        if vm_running():
            virsh("destroy", VM_NAME)
        virsh("undefine", VM_NAME, "--snapshots-metadata")
    pool_vol_delete(DISK_VOL)
    pool_vol_delete(SEED_VOL)
    if getattr(args, "purge_base", False):
        pool_vol_delete(BASE_VOL)
        shutil.rmtree(STATE_DIR / "base", ignore_errors=True)
        log("base image purged")
    log(f"VM {VM_NAME!r} destroyed")

def cmd_recreate(args):
    cmd_destroy(args)
    cmd_create(None)

def cmd_status(_args):
    if not vm_exists():
        print(f"{VM_NAME}: not created")
        return
    print(f"name:      {VM_NAME}\nstate:     {vm_state()}\nip:        {safe_vm_ip()}\nuri:       {LIBVIRT_URI}\nartifacts: {ARTIFACTS_ROOT / 'latest'}")
    print("snapshots:")
    for name in virsh("snapshot-list", VM_NAME, "--name").splitlines():
        if name.strip():
            print(f"  - {name.strip()}")

def safe_vm_ip():
    try: return vm_ip()
    except VMError: return "n/a"

def cmd_ip(_args):
    print(vm_ip())

def cmd_ssh(args):
    ensure_running()
    subprocess.run(ssh_args(args.remote), check=True)

def cmd_console(_args):
    run(["virsh", "-c", LIBVIRT_URI, "console", VM_NAME], capture=False)

def cmd_viewer(_args):
    if not have("virt-viewer"):
        raise VMError("virt-viewer is not installed")
    if VM_GRAPHICS == "none":
        raise VMError("VM was created without graphics")
    subprocess.Popen(["virt-viewer", "-c", LIBVIRT_URI, VM_NAME],
                     stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                     start_new_session=True)

def cmd_gui(args):
    ensure_running()
    mode = args.mode or "on"
    if mode == "on":
        guest("sudo touch /etc/i3-next-gui-session && sudo systemctl restart getty@tty1")
        cmd_viewer(None)
    elif mode == "off":
        guest("sudo rm -f /etc/i3-next-gui-session && sudo systemctl restart getty@tty1")
    else:
        raise VMError("usage: vm.py gui [on|off]")

def cmd_screenshot(args):
    output = Path(args.file or (STATE_DIR / "screenshots" / f"{dt.datetime.now():%Y%m%d-%H%M%S}.png"))
    output.parent.mkdir(parents=True, exist_ok=True)
    virsh("screenshot", VM_NAME, "--file", output)
    print(output)

def cmd_sync(_args):
    ensure_running()
    ensure_provisioned()
    ip = vm_ip()
    log(f"syncing repository to guest:~/{'i3-next'}")
    run(["rsync", "-az", "--delete", "--filter=:- .gitignore",
         "--exclude=.git/", "--exclude=/tests/vm/.state/",
         "-e", rsync_shell(), f"{REPO_ROOT}/", f"{VM_USER}@{ip}:i3-next/"])

def cmd_build(_args):
    ensure_running()
    log("building inside the guest")
    guest("cd ~/i3-next && make build")

def cmd_test(_args):
    if setting("VM_RESTORE", "0") == "1":
        cmd_restore(argparse.Namespace(name=SNAPSHOT_NAME))
    ensure_running()
    cmd_sync(None)
    cmd_build(None)
    remote = "/tmp/i3-next-artifacts"
    guest(f"rm -rf {remote} && mkdir -p {remote}")
    log("running end-to-end suite")
    rc = guest(f"cd ~/i3-next && ARTIFACT_DIR={remote} OUTPUTS={setting('OUTPUTS', '2')} bash tests/e2e/run.sh", check=False)
    stamp = f"{dt.datetime.now():%Y%m%d-%H%M%S}"
    dest = ARTIFACTS_ROOT / stamp
    dest.mkdir(parents=True, exist_ok=True)
    try:
        run(["rsync", "-az", "-e", rsync_shell(), f"{VM_USER}@{vm_ip()}:{remote}/", f"{dest}/"])
    except VMError:
        warn("could not fetch artifacts")
    latest = ARTIFACTS_ROOT / "latest"
    latest.unlink(missing_ok=True)
    latest.symlink_to(stamp)
    log(f"artifacts: {dest}")
    if rc:
        raise VMError(f"E2E suite failed with exit code {rc}")

def cmd_snapshot(args):
    name = args.name or SNAPSHOT_NAME
    if not vm_exists():
        raise VMError("VM does not exist")
    was_running = vm_running()
    if was_running:
        cmd_stop(None)
    virsh("snapshot-delete", VM_NAME, name, check=False)
    virsh("snapshot-create-as", VM_NAME, name,
          "--description", "i3-next dependencies provisioned baseline")
    log(f"snapshot {name!r} created")
    if was_running:
        cmd_start(None)

def cmd_restore(args):
    name = args.name or SNAPSHOT_NAME
    if not vm_exists():
        raise VMError("VM does not exist")
    if vm_running():
        virsh("destroy", VM_NAME)
    virsh("snapshot-revert", VM_NAME, name)
    log(f"reverted to snapshot {name!r}")
    cmd_start(None)

def cmd_snapshots(_args):
    virsh("snapshot-list", VM_NAME, capture=False)

def cmd_nic(args):
    if args.state not in ("up", "down"):
        raise VMError("usage: vm.py nic up|down")
    rows = virsh("domiflist", VM_NAME).splitlines()
    dev = next((line.split()[0] for line in rows[2:] if line.split()), None)
    if not dev:
        raise VMError("no network interface found")
    virsh("domif-setlink", VM_NAME, dev, args.state)
    log(f"link {dev} -> {args.state}")

def cmd_artifacts(_args):
    latest = ARTIFACTS_ROOT / "latest"
    print(latest)
    if latest.exists():
        for item in sorted(latest.iterdir()):
            print(item.name)

def build_parser():
    parser = argparse.ArgumentParser(description=f"i3-next KVM lab (VM: {VM_NAME}, libvirt: {LIBVIRT_URI})")
    sub = parser.add_subparsers(dest="command")
    commands = {
        "doctor": cmd_doctor, "image": cmd_image, "create": cmd_create,
        "provision": cmd_provision, "up": cmd_up, "start": cmd_start,
        "stop": cmd_stop, "status": cmd_status, "ip": cmd_ip,
        "build": cmd_build, "test": cmd_test, "sync": cmd_sync,
        "logs": cmd_logs, "viewer": cmd_viewer, "snapshots": cmd_snapshots,
        "artifacts": cmd_artifacts,
    }
    for name in commands:
        p = sub.add_parser(name)
        p.set_defaults(func=commands[name])
    p = sub.add_parser("destroy")
    p.add_argument("--purge-base", action="store_true")
    p.set_defaults(func=cmd_destroy)
    p = sub.add_parser("recreate")
    p.add_argument("--purge-base", action="store_true")
    p.set_defaults(func=cmd_recreate)
    for name, fn in (("ssh", cmd_ssh), ("console", cmd_console),
                     ("viewer", cmd_viewer), ("gui", cmd_gui),
                     ("screenshot", cmd_screenshot), ("snapshot", cmd_snapshot),
                     ("restore", cmd_restore), ("nic", cmd_nic)):
        if name in ("viewer",):
            continue
        p = sub.add_parser(name)
        if name == "ssh":
            p.add_argument("remote", nargs=argparse.REMAINDER)
        elif name == "gui":
            p.add_argument("mode", nargs="?", choices=("on", "off"), default="on")
        elif name == "screenshot":
            p.add_argument("file", nargs="?")
        elif name in ("snapshot", "restore"):
            p.add_argument("name", nargs="?")
        elif name == "nic":
            p.add_argument("state", choices=("up", "down"))
        p.set_defaults(func=fn)
    return parser

def main():
    parser = build_parser()
    args = parser.parse_args()
    if not args.command:
        parser.print_help()
        return 0
    try:
        args.func(args)
        return 0
    except (VMError, OSError, subprocess.SubprocessError) as exc:
        print(f"[error] {exc}", file=sys.stderr)
        return 1

if __name__ == "__main__":
    raise SystemExit(main())
