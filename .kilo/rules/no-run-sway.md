IMPORTANT constraint for this repo (i3_next / i3-next):

Do NOT start, run, or test against Sway on the host machine.
- Never run `sway`, `swaymsg`, headless Sway (WLR_BACKENDS=headless), or any e2e script that launches Sway directly on the host.
- Do not run `tests/e2e/run.sh` on the host, and do not execute `make vm-test` / `tests/vm/vm.py test` expecting results outside the VM.
- All Sway-dependent verification (e2e, headless outputs, hotplug, screenshots, daemon↔Sway integration, reconnect tests) MUST happen inside the KVM guest via the VM lab: `make vm-test`, `make vm-test-clean`, `make vm-gui`, `make vm-restart`.
- On the host you may run only: `go build ./...`, `go test -race ./...` (unit tests that do NOT require a compositor), `gofmt`, `go vet`, `golangci-lint`, `staticcheck`, `scripts/check-naming.sh`, and `make check` minus anything that spins up Sway.
- If a task appears to require Sway and the VM is unavailable, stop and report it as blocked instead of falling back to the host.
- Reason: the host is not a Wayland/Sway session; running Sway here is unsafe, may steal input/display, and produces false results. The VM (Debian 13 guest with headless Sway) is the only supported test bed for compositor-level behavior.