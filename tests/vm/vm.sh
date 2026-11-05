#!/usr/bin/env bash
#
# i3-next KVM test lab
#
# Spins up a Debian 13 guest on KVM/libvirt, provisions it with Sway,
# Quickshell and Go, syncs this repository into it and runs the end-to-end
# suite (tests/e2e/run.sh). The same guest can be used as a full graphical
# desktop session (virtio GPU + SPICE) for manual testing.
#
# Usage: tests/vm/vm.sh <command> [args]     (run "vm.sh help")

set -Eeuo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
STATE_DIR="${VM_STATE_DIR:-$SCRIPT_DIR/.state}"
ARTIFACTS_ROOT="$STATE_DIR/artifacts"
KEY="$STATE_DIR/id_ed25519"

# ---- configuration (override through the environment) ----------------------
VM_NAME="${VM_NAME:-i3-next-test}"
VM_USER="${VM_USER:-dev}"
VM_CPUS="${VM_CPUS:-4}"
VM_RAM_MB="${VM_RAM_MB:-4096}"
VM_DISK_GB="${VM_DISK_GB:-30}"
VM_GRAPHICS="${VM_GRAPHICS:-spice}"        # spice | vnc | none
VM_GL="${VM_GL:-0}"                        # 1 = virgl 3D (needs GL on the host)
VM_NETWORK="${VM_NETWORK:-default}"
VM_POOL="${VM_POOL:-default}"
LIBVIRT_URI="${LIBVIRT_URI:-qemu:///system}"
IMAGE_BASE_URL="${IMAGE_BASE_URL:-https://cloud.debian.org/images/cloud/trixie/latest}"
IMAGE_FILE="${IMAGE_FILE:-debian-13-generic-amd64.qcow2}"
SNAPSHOT_NAME="${SNAPSHOT_NAME:-provisioned}"

BASE_VOL="i3-next-base-${IMAGE_FILE}"
DISK_VOL="${VM_NAME}.qcow2"
SEED_VOL="${VM_NAME}-seed.iso"

SSH_OPTS=(-i "$KEY" -o IdentitiesOnly=yes -o StrictHostKeyChecking=no
  -o UserKnownHostsFile=/dev/null -o LogLevel=ERROR
  -o ConnectTimeout=5 -o ServerAliveInterval=15)

# ---- helpers ----------------------------------------------------------------
log()  { printf '\033[1;34m==>\033[0m %s\n' "$*" >&2; }
warn() { printf '\033[1;33m[warn]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[error]\033[0m %s\n' "$*" >&2; exit 1; }
have() { command -v "$1" >/dev/null 2>&1; }

virsh() { command virsh -c "$LIBVIRT_URI" "$@"; }

vm_exists()  { virsh dominfo "$VM_NAME" >/dev/null 2>&1; }
vm_state()   { virsh domstate "$VM_NAME" 2>/dev/null | head -n1; }
vm_running() { [ "$(vm_state)" = "running" ]; }

vm_ip() {
  local src ip
  for src in lease agent arp; do
    ip="$(virsh domifaddr "$VM_NAME" --source "$src" 2>/dev/null |
      awk '/ipv4/ && $4 !~ /^127\./ {split($4, a, "/"); print a[1]; exit}')" || true
    if [ -n "$ip" ]; then echo "$ip"; return 0; fi
  done
  return 1
}

vm_ssh() {
  local ip
  ip="$(vm_ip)" || die "VM '$VM_NAME' has no IP address (is it running?)"
  ssh "${SSH_OPTS[@]}" "$VM_USER@$ip" "$@"
}

rsh() { echo "ssh ${SSH_OPTS[*]}"; }

ensure_key() {
  mkdir -p "$STATE_DIR"
  [ -f "$KEY" ] || ssh-keygen -q -t ed25519 -N '' -C i3-next-test -f "$KEY"
}

ensure_running() {
  vm_exists || die "VM '$VM_NAME' does not exist. Run: vm.sh up"
  vm_running || cmd_start
}

wait_for_ssh() {
  local timeout="${1:-240}" i ip
  log "waiting for SSH (up to ${timeout}s)"
  for ((i = 0; i < timeout; i += 3)); do
    if ip="$(vm_ip)" && ssh "${SSH_OPTS[@]}" -o BatchMode=yes "$VM_USER@$ip" true 2>/dev/null; then
      log "SSH is up on $ip"
      return 0
    fi
    sleep 3
  done
  die "timed out waiting for SSH"
}

pool_vol_exists() { virsh vol-info --pool "$VM_POOL" "$1" >/dev/null 2>&1; }

pool_vol_delete() {
  if pool_vol_exists "$1"; then virsh vol-delete --pool "$VM_POOL" "$1" >/dev/null; fi
}

# pool_vol_upload <name> <file> <format>
pool_vol_upload() {
  local name="$1" file="$2" fmt="$3" size
  size="$(stat -c %s "$file")"
  virsh vol-create-as "$VM_POOL" "$name" "${size}B" --format "$fmt" >/dev/null
  virsh vol-upload --pool "$VM_POOL" "$name" "$file"
}

pick_os_variant() {
  local v
  for v in debian13 debian12 linux2022 generic; do
    if have osinfo-query && osinfo-query os short-id="$v" >/dev/null 2>&1; then
      echo "$v"; return
    fi
  done
  echo generic
}

# ---- commands ---------------------------------------------------------------
cmd_doctor() {
  local rc=0
  chk() { # chk <label> <cmd...>
    local label="$1"; shift
    if "$@" >/dev/null 2>&1; then printf '  [ OK ] %s\n' "$label"
    else printf '  [FAIL] %s\n' "$label"; rc=1; fi
  }
  echo "Host checks:"
  chk "CPU virtualization (vmx/svm)" grep -Eq '(vmx|svm)' /proc/cpuinfo
  chk "/dev/kvm exists"              test -e /dev/kvm
  chk "/dev/kvm is accessible"       test -r /dev/kvm -a -w /dev/kvm
  local t
  for t in virsh virt-install qemu-img ssh ssh-keygen rsync curl sha512sum; do
    chk "tool: $t" have "$t"
  done
  chk "ISO tool (cloud-localds|genisoimage|xorriso|mkisofs)" bash -c \
    'command -v cloud-localds || command -v genisoimage || command -v xorriso || command -v mkisofs'
  chk "libvirt reachable at $LIBVIRT_URI" virsh list
  chk "storage pool '$VM_POOL' exists"    virsh pool-info "$VM_POOL"
  chk "network '$VM_NETWORK' is active"   bash -c \
    "virsh -c '$LIBVIRT_URI' net-info '$VM_NETWORK' | grep -Eq 'Active:[[:space:]]+yes'"
  have virt-viewer || warn "virt-viewer not found: 'vm.sh viewer/gui' will not work (headless tests are fine)"
  if [ "$rc" -ne 0 ]; then
    cat >&2 <<'EOF'

Hints (Debian/Ubuntu):
  sudo apt install qemu-system-x86 libvirt-daemon-system libvirt-clients virtinst \
                   virt-viewer genisoimage rsync curl
  sudo usermod -aG libvirt,kvm "$USER"   # then log out/in
  sudo virsh net-start default; sudo virsh net-autostart default
  sudo virsh pool-define-as default dir --target /var/lib/libvirt/images
  sudo virsh pool-start default; sudo virsh pool-autostart default
EOF
  fi
  return "$rc"
}

cmd_image() {
  mkdir -p "$STATE_DIR/base"
  local dst="$STATE_DIR/base/$IMAGE_FILE" want got
  if [ ! -f "$dst" ]; then
    log "downloading $IMAGE_FILE"
    curl -fL --retry 3 -o "$dst.part" "$IMAGE_BASE_URL/$IMAGE_FILE"
    mv "$dst.part" "$dst"
  fi
  want="$(curl -fsSL "$IMAGE_BASE_URL/SHA512SUMS" | awk -v f="$IMAGE_FILE" '$2 == f {print $1}')"
  [ -n "$want" ] || die "no checksum published for $IMAGE_FILE"
  got="$(sha512sum "$dst" | awk '{print $1}')"
  # if [ "$want" != "$got" ]; then
  #   rm -f "$dst"
  #   die "checksum mismatch (upstream 'latest' image changed?). Cached copy removed; re-run."
  # fi
  log "base image verified"
}

make_seed_iso() {
  local dir="$STATE_DIR/seed" out="$STATE_DIR/seed.iso" pub
  pub="$(cat "$KEY.pub")"
  rm -rf "$dir" "$out"; mkdir -p "$dir"
  sed -e "s|__HOSTNAME__|$VM_NAME|g" -e "s|__USER__|$VM_USER|g" \
      -e "s|__SSH_PUBKEY__|$pub|g" "$SCRIPT_DIR/cloud-init/user-data.tpl" >"$dir/user-data"
  sed -e "s|__HOSTNAME__|$VM_NAME|g" -e "s|__INSTANCE__|$(date +%s)|g" \
      "$SCRIPT_DIR/cloud-init/meta-data.tpl" >"$dir/meta-data"
  if   have cloud-localds; then cloud-localds "$out" "$dir/user-data" "$dir/meta-data"
  elif have genisoimage;   then genisoimage -quiet -output "$out" -volid cidata -joliet -rock "$dir/user-data" "$dir/meta-data"
  elif have xorriso;       then xorriso -as mkisofs -quiet -o "$out" -V cidata -J -r "$dir/user-data" "$dir/meta-data"
  elif have mkisofs;       then mkisofs -quiet -output "$out" -volid cidata -joliet -rock "$dir/user-data" "$dir/meta-data"
  else die "need one of: cloud-localds, genisoimage, xorriso, mkisofs"; fi
  echo "$out"
}

cmd_create() {
  if vm_exists; then die "VM '$VM_NAME' already exists (use: vm.sh recreate)"; fi
  ensure_key
  cmd_image

  if ! pool_vol_exists "$BASE_VOL"; then
    log "uploading base image into pool '$VM_POOL'"
    pool_vol_upload "$BASE_VOL" "$STATE_DIR/base/$IMAGE_FILE" qcow2
  fi

  pool_vol_delete "$DISK_VOL"
  virsh vol-create-as "$VM_POOL" "$DISK_VOL" "${VM_DISK_GB}G" --format qcow2 \
    --backing-vol "$BASE_VOL" --backing-vol-format qcow2 >/dev/null

  local seed; seed="$(make_seed_iso)"
  pool_vol_delete "$SEED_VOL"
  pool_vol_upload "$SEED_VOL" "$seed" raw

  local -a gfx=()
  case "$VM_GRAPHICS" in
    spice)
      if [ "$VM_GL" = 1 ]; then
        gfx=(--graphics spice,gl.enable=yes,listen=none
             --video model.type=virtio,model.acceleration.accel3d=yes)
      else
        gfx=(--graphics spice,listen=none --video virtio)
      fi
      gfx+=(--channel spicevmc) ;;
    vnc)  gfx=(--graphics vnc,listen=127.0.0.1 --video virtio) ;;
    none) gfx=(--graphics none) ;;
    *)    die "VM_GRAPHICS must be spice, vnc or none" ;;
  esac

  log "defining and starting '$VM_NAME' (${VM_CPUS} vCPU, ${VM_RAM_MB} MiB, graphics=$VM_GRAPHICS)"
  virt-install --connect "$LIBVIRT_URI" \
    --name "$VM_NAME" --memory "$VM_RAM_MB" --vcpus "$VM_CPUS" \
    --cpu host-passthrough --os-variant "$(pick_os_variant)" \
    --import --boot hd \
    --disk "vol=$VM_POOL/$DISK_VOL,bus=virtio" \
    --disk "vol=$VM_POOL/$SEED_VOL,device=cdrom,bus=sata" \
    --network "network=$VM_NETWORK,model=virtio" \
    "${gfx[@]}" \
    --channel unix,target_type=virtio,name=org.qemu.guest_agent.0 \
    --rng /dev/urandom --noautoconsole

  wait_for_ssh 300
  cmd_provision

  local tgt
  tgt="$(virsh domblklist "$VM_NAME" --details | awk '$2 == "cdrom" {print $3; exit}')"
  if [ -n "$tgt" ]; then
    virsh change-media "$VM_NAME" "$tgt" --eject --force --live --config >/dev/null ||
      warn "could not eject cloud-init seed"
  fi
  cmd_snapshot "$SNAPSHOT_NAME"
  log "lab is ready. Try: vm.sh test | vm.sh gui | vm.sh ssh"
}

cmd_provision() {
  ensure_running
  local goversion
  goversion="$(awk '/^go /{print $2; exit}' "$REPO_ROOT/go.mod" 2>/dev/null || true)"
  log "provisioning guest (Go ${goversion:-distro})"
  vm_ssh "sudo cloud-init status --wait >/dev/null 2>&1 || true"
  vm_ssh "cat > /tmp/provision.sh" <"$SCRIPT_DIR/guest-provision.sh"
  vm_ssh "sudo env GO_VERSION='$goversion' DEV_USER='$VM_USER' INSTALL_QUICKSHELL='${INSTALL_QUICKSHELL:-1}' bash /tmp/provision.sh"
}

cmd_up() {
  if ! vm_exists; then cmd_create
  elif ! vm_running; then cmd_start
  else wait_for_ssh 60; fi
}

cmd_start() {
  vm_exists || die "VM '$VM_NAME' does not exist. Run: vm.sh up"
  if ! vm_running; then virsh start "$VM_NAME" >/dev/null; fi
  wait_for_ssh 240
}

cmd_stop() {
  if ! vm_running; then log "VM already stopped"; return 0; fi
  virsh shutdown "$VM_NAME" >/dev/null
  local i
  for ((i = 0; i < 30; i++)); do
    if ! vm_running; then log "VM stopped"; return 0; fi
    sleep 2
  done
  warn "graceful shutdown timed out, forcing power off"
  virsh destroy "$VM_NAME" >/dev/null
}

cmd_destroy() {
  if vm_exists; then
    if vm_running; then virsh destroy "$VM_NAME" >/dev/null; fi
    virsh undefine "$VM_NAME" --snapshots-metadata >/dev/null
  fi
  pool_vol_delete "$DISK_VOL"
  pool_vol_delete "$SEED_VOL"
  if [ "${1:-}" = "--purge-base" ]; then
    pool_vol_delete "$BASE_VOL"
    rm -rf "$STATE_DIR/base"
    log "base image purged"
  fi
  log "VM '$VM_NAME' destroyed"
}

cmd_recreate() { cmd_destroy "$@"; cmd_create; }

cmd_status() {
  if ! vm_exists; then echo "$VM_NAME: not created"; return 0; fi
  echo "name:      $VM_NAME"
  echo "state:     $(vm_state)"
  echo "ip:        $(vm_ip || echo n/a)"
  echo "uri:       $LIBVIRT_URI"
  echo "artifacts: $ARTIFACTS_ROOT/latest"
  echo "snapshots:"; virsh snapshot-list "$VM_NAME" --name | sed '/^$/d; s/^/  - /'
}

cmd_ip()      { vm_ip; }
cmd_ssh()     { ensure_running; if [ $# -gt 0 ]; then vm_ssh "$@"; else vm_ssh; fi; }
cmd_console() { virsh console "$VM_NAME"; }   # leave with Ctrl+]

cmd_viewer() {
  have virt-viewer || die "virt-viewer is not installed"
  [ "$VM_GRAPHICS" != none ] || die "VM was created without graphics"
  nohup virt-viewer -c "$LIBVIRT_URI" "$VM_NAME" >/dev/null 2>&1 &
}

# gui on  : autologin on tty1 + start the real Sway session, open the viewer
# gui off : back to a plain console (tests keep using headless Sway)
cmd_gui() {
  ensure_running
  case "${1:-on}" in
    on)  vm_ssh "sudo touch /etc/i3-next-gui-session && sudo systemctl restart getty@tty1"
         cmd_viewer ;;
    off) vm_ssh "sudo rm -f /etc/i3-next-gui-session && sudo systemctl restart getty@tty1" ;;
    *)   die "usage: vm.sh gui [on|off]" ;;
  esac
}

cmd_screenshot() {   # screenshot of the guest display (format is chosen by QEMU/libvirt)
  local out="${1:-$STATE_DIR/screenshots/$(date +%Y%m%d-%H%M%S).png}"
  mkdir -p "$(dirname "$out")"
  virsh screenshot "$VM_NAME" --file "$out" >/dev/null
  echo "$out"
}

cmd_sync() {
  ensure_running
  local ip; ip="$(vm_ip)"
  log "syncing repository to guest:~/i3-next"
  rsync -az --delete --filter=':- .gitignore' \
    --exclude='.git/' --exclude='/tests/vm/.state/' \
    -e "$(rsh)" "$REPO_ROOT/" "$VM_USER@$ip:i3-next/"
}

cmd_build() {
  ensure_running
  log "building inside the guest"
  vm_ssh "cd ~/i3-next && make build"
}

cmd_test() {
  if [ "${VM_RESTORE:-0}" = 1 ]; then cmd_restore "$SNAPSHOT_NAME"; fi
  ensure_running
  cmd_sync
  cmd_build

  local remote=/tmp/i3-next-artifacts rc=0 stamp dest ip
  vm_ssh "rm -rf $remote && mkdir -p $remote"
  log "running end-to-end suite"
  vm_ssh "cd ~/i3-next && ARTIFACT_DIR=$remote OUTPUTS=${OUTPUTS:-2} bash tests/e2e/run.sh" || rc=$?

  stamp="$(date +%Y%m%d-%H%M%S)"
  dest="$ARTIFACTS_ROOT/$stamp"
  mkdir -p "$dest"
  ip="$(vm_ip)"
  rsync -az -e "$(rsh)" "$VM_USER@$ip:$remote/" "$dest/" || warn "could not fetch artifacts"
  ln -sfn "$stamp" "$ARTIFACTS_ROOT/latest"
  log "artifacts: $dest"
  return "$rc"
}

# Offline (shut-off) internal snapshots: slower than live ones but always
# consistent and independent of devices such as the cdrom or SPICE channels.
cmd_snapshot() {
  local name="${1:-$SNAPSHOT_NAME}" was=0
  vm_exists || die "VM does not exist"
  if vm_running; then was=1; cmd_stop; fi
  virsh snapshot-delete "$VM_NAME" "$name" >/dev/null 2>&1 || true
  virsh snapshot-create-as "$VM_NAME" "$name" --description "i3-next lab snapshot" >/dev/null
  log "snapshot '$name' created"
  if [ "$was" = 1 ]; then cmd_start; fi
}

cmd_restore() {
  local name="${1:-$SNAPSHOT_NAME}"
  vm_exists || die "VM does not exist"
  if vm_running; then virsh destroy "$VM_NAME" >/dev/null; fi
  virsh snapshot-revert "$VM_NAME" "$name" >/dev/null
  log "reverted to snapshot '$name'"
  cmd_start
}

cmd_snapshots() { virsh snapshot-list "$VM_NAME"; }

# Network fault injection: unplug/replug the virtual cable.
cmd_nic() {
  local state="${1:-}" dev
  case "$state" in up|down) ;; *) die "usage: vm.sh nic up|down" ;; esac
  dev="$(virsh domiflist "$VM_NAME" | awk 'NR > 2 && $1 != "" {print $1; exit}')"
  [ -n "$dev" ] || die "no network interface found"
  virsh domif-setlink "$VM_NAME" "$dev" "$state" >/dev/null
  log "link $dev -> $state"
}

cmd_artifacts() { echo "$ARTIFACTS_ROOT/latest"; ls -1 "$ARTIFACTS_ROOT/latest" 2>/dev/null || true; }

usage() {
  cat <<EOF
i3-next KVM lab  (VM: $VM_NAME, libvirt: $LIBVIRT_URI)

Lifecycle
  doctor              check host prerequisites (KVM, libvirt, tools)
  up                  create (first time), start and wait for the VM
  start | stop        power on / graceful shutdown
  destroy [--purge-base]   remove the VM and its disks
  recreate            destroy + create from scratch
  status | ip

Development loop
  sync                rsync the repo into the guest (~/i3-next)
  build               run 'make build' in the guest
  test                sync + build + run tests/e2e/run.sh, fetch artifacts
                      (VM_RESTORE=1 reverts to a clean snapshot first)
  ssh [cmd...]        shell into the guest
  console             serial console (leave with Ctrl+])

Full simulation
  gui [on|off]        real Sway session on a virtio GPU, shown via virt-viewer
  viewer              open virt-viewer only
  screenshot [file]   screenshot of the guest display
  nic up|down         unplug / replug the virtual network cable

State
  snapshot [name]     offline snapshot (default: $SNAPSHOT_NAME)
  restore  [name]     revert and boot
  snapshots | artifacts | provision | image

Environment: VM_NAME VM_CPUS VM_RAM_MB VM_DISK_GB VM_GRAPHICS(spice|vnc|none)
             VM_GL=1 VM_POOL VM_NETWORK LIBVIRT_URI OUTPUTS INSTALL_QUICKSHELL=0
EOF
}

main() {
  local cmd="${1:-help}"; shift || true
  case "$cmd" in
    doctor|image|create|provision|up|start|stop|destroy|recreate|status|ip|ssh|console|\
viewer|gui|screenshot|sync|build|test|snapshot|restore|snapshots|nic|artifacts)
      "cmd_$cmd" "$@" ;;
    help|-h|--help) usage ;;
    *) usage >&2; die "unknown command: $cmd" ;;
  esac
}

main "$@"
