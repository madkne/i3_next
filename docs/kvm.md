# i3-next KVM lab

Full-system test bed: a Debian 13 guest on KVM/libvirt with Sway, Quickshell,
Go and the Wayland tooling. Two ways to use it:

* **Automated** – `make vm-test` syncs the repo, builds it in the guest and runs
  `tests/e2e/run.sh` against a *headless* Sway with N virtual monitors.
  Logs, screenshots and Sway state land in `tests/vm/.state/artifacts/latest/`.
* **Interactive** – `make vm-gui` starts the real session (autologin on tty1,
  virtio GPU) and opens it in virt-viewer.

## Setup

    tests/vm/vm.sh doctor     # tells you what is missing
    make vm-up                # first run downloads the image and provisions (~5-10 min)

Add `-include tests/vm/vm.mk` to the Makefile and `tests/vm/.state/` to `.gitignore`.

## Everyday loop

    make vm-test              # incremental, ~1 min
    make vm-test-clean        # reverts to the "provisioned" snapshot first
    tests/vm/vm.sh ssh        # poke around
    tests/vm/vm.sh nic down   # simulate a network outage
    tests/vm/vm.sh screenshot # grab the guest display

Tunables (env): `VM_CPUS VM_RAM_MB VM_DISK_GB VM_GRAPHICS=spice|vnc|none VM_GL=1
OUTPUTS=2 LIBVIRT_URI=qemu:///session INSTALL_QUICKSHELL=0`.

## Adding tests

Write a `t_<name>` function in `tests/e2e/run.sh` (it runs with `set -e`, output
is kept per test) and register it with `run_test <name> t_<name>`. Sway is
already running with `$SWAYSOCK`/`$WAYLAND_DISPLAY` exported, so `swaymsg`,
`grim`, `wtype`, `foot` and the future `i3-nextctl` can be used directly.

## Simulating more hardware

* Battery/AC: `sudo modprobe test_power` in the guest (fake `power_supply`).
* Extra monitors: `swaymsg create_output` (headless) or `OUTPUTS=N`.
* Network: `vm.sh nic up|down`, or `virsh attach-interface` for hotplug.
* Bluetooth/audio: `modprobe vhci-hcd` / add `--sound` to `virt-install`.
* Real 3D: `VM_GL=1 vm.sh recreate` (needs OpenGL on the host).

## Notes

* Base image is cached in `.state/base` and uploaded to the libvirt pool, so
  qemu never needs access to your home directory.
* Snapshots are taken offline (VM shut down) for consistency.
* CI: any runner with `/dev/kvm` (self-hosted or GitHub's larger Linux runners)
  can run `make vm-up vm-test`.
