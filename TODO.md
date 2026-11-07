# i3_next — Production v1 Release TODO

> Goal: take the repo from today's state (daemon skeleton, basic CLI, QML stubs) to a first version that people can install, use, and get support for.
>
> **Definition of Done:** an item is checked only when: 1) the code is written, 2) related tests (unit/integration/e2e) pass in CI, 3) the needed docs are written, 4) `make check` and `make vm-test` pass.
>
> **Legend**
>
> | Mark | Meaning |
> |---|---|
> | `P0` | Release blocker |
> | `P1` | Expected in v1 |
> | `P2` | Can move to v1.x if time is short |
> | `⛓ X` | Depends on task X |
> | `✅` | Testable acceptance criteria |
>
> ID prefixes: `FIX` quick fixes, `CORE`, `CFG`, `SWAY`, `API`, `UI`, `SVC`, `DBUS`, `PLG`, `STO`, `SEC`, `OBS`, `SES`, `PKG`, `QA`, `PERF`, `DOC`, `CI`.

---

## 0. Scope and release criteria (done)

- [x] v1 scope, target distros, architecture, compatibility matrix, minimum hardware, session model (full Sway session), must-have list, out-of-scope list, acceptance criteria per feature, and versioning policy are written in `docs/RELEASE_SCOPE.md`.
- [x] No README feature is described as existing unless it is built (see "Planned Features").

### Reopened items

- [x] **FIX-00a** (P0) `RELEASE_SCOPE.md` does not match `go.mod`: it says minimum Go is 1.24, but `go.mod` says `1.27.3`. ✅ Pick one version and make `go.mod`, CI, `CONTRIBUTING.md`, and `RELEASE_SCOPE.md` agree. (⛓ FIX-05)

---

## 1. Project structure and base quality

### Done
- [x] Shared contracts (`internal/api/contracts`), domain errors, `LICENSE`, `CONTRIBUTING.md`, `SECURITY.md`, pre-commit, basic CI, `.golangci.yml`.

### Reopened / quick fixes (all P0 unless noted)

- [x] **FIX-01** Plain `make` runs `vm-doctor` instead of `all` (because `-include tests/vm/vm.mk` comes before `all`). ✅ Add `.DEFAULT_GOAL := all` and test that `make -n` picks `all`. Also replace `$(GO)fmt` with `GOFMT := gofmt`.
- [x] **FIX-02** `lib.sh` is in the repo root, but `tests/e2e/run.sh` sources `tests/e2e/lib.sh`. ✅ Move the file to `tests/e2e/lib.sh`; `make vm-test` runs with no source error.
- [x] **FIX-03** `configs/sway/config` includes `$HOME/.config/i3-next/sway/*`, which nothing installs, and the keybindings call commands that do not exist (`i3-nextctl launcher`, `i3-nextctl session logout`). ✅
  - Make the include path relative to the config file (or install into `~/.config/i3-next/sway/`), and make `sway --validate` pass on the installed path in e2e.
  - Until the commands exist, change the missing binds to `exec foot` / `swaymsg exit`, or comment them out.
- [x] **FIX-04** The version is repeated in many places and does not match (`i3-nextd` = `0.1.1`, others `0.1.0`, `app.go` has a hard-coded `"0.1.0"`, plus `contracts.AppVersion`). ✅ Create an `internal/version` package (Version/Commit/BuildDate from `-ldflags`); delete the `info.go` files; all binaries (including `i3-next-doctor`) give the same `--version` output; e2e `t_version_flags` covers all four binaries.
- [x] **FIX-05** Check the Go version. ✅ `go.mod` has only `go` (the real minimum) and `toolchain` if needed; CI uses `go-version-file: go.mod`; the chosen version can be installed on both Tier-1 distros (`guest-provision.sh` already checks with HEAD).
- [x] **FIX-07** `bootstrapDirs` creates directories with `0o755`, but `SECURITY.md` promises minimal permissions. ✅ Create runtime/state/data/config with `0o700`; add a unit test for the mode (using `t.TempDir()` and `XDG_*`).
- [x] **FIX-08** Removing the lock file after `flock(UN)` has a classic race (a second process can lock the deleted inode). ✅ Do not delete the lock file (or re-check the inode with `stat` after open); test with two processes at once where exactly one wins.
- [x] **FIX-09** `internal/app/bootstrap`: ✅
  - Replace the reflection-based `callLifecycle` with typed interfaces (`Starter`, `Stopper`, `HealthChecker`, `Reloader`, `Named`). (The current `param.Implements(ctxType)` check is backwards.)
  - Remove the aliases `AddService/Add/Use`.
  - Merge `ErrAlreadyRunning/ErrNotRunning` with `internal/core/errors`; keep `ErrLoggerRequired` in one place.
- [x] **FIX-10** `App.Run`: ✅
  - Log and collect `Stop` errors (do not ignore them).
  - Log "starting" after startup succeeds, and take the version from `internal/version`.
  - Watch for service failure while running (Health/exit) and go degraded or shut down by policy.
  - Tests: rollback on start failure, reverse stop order, shutdown timeout, hook failure, running again after Run.
- [x] **FIX-12** (P0) CI: ✅
  - Pin actions by SHA; pin `staticcheck` and `golangci-lint` to exact versions (not `latest`).
  - Add `govulncheck ./...`.
  - Add `concurrency: cancel-in-progress` and `timeout-minutes` to jobs.
  - The QML job uses `|| true`, so it is always green; it must really install `qmllint` or fail.
  - Add a Go version matrix (minimum and current).
- [x] **FIX-13** The README "Structure" section is the target layout, not the real one. ✅ Title it "Target structure (not yet created)" or show the real tree; also set `[x]` items in section 1 that are only skeletons (package layout, empty config/QML files) back to `[ ]`.
- [x] **FIX-14** (P1) `SECURITY.md` points to `docs/SECURITY.md`, which does not exist; `scripts/build.sh` is a second build source; `scripts/install.sh` installs only binaries (not config/QML/session). ✅ Make a final decision and align them. (⛓ PKG-01)
- [x] **FIX-15** `i3-next-doctor` always exits 0, even with `[FAIL]`. ✅ Non-zero exit on FAIL, `--json`, and a new e2e test. (⛓ OBS-04)
- [x] **FIX-16** (P1) One naming rule: `i3_next` (module/AppName) vs `i3-next` (dirs/binaries). ✅ Write it in `docs/development/naming.md` and add a script that checks it.

### Rest of section 1
- [x] **STRUCT-01** (P0) Create packages only when they have real code (no empty folders). ✅ Enforce the import rule (core ⟵ adapters/services ⟵ api) with `depguard` / `go-arch-lint` in `.golangci.yml`.
- [x] **STRUCT-02** (P1) `CODEOWNERS`, PR/Issue templates, `CHANGELOG.md` (Keep a Changelog).
- [x] **STRUCT-03** (P1) `go.sum` and a dependency policy: every new dependency needs a reason in the PR and a license check. (⛓ SEC-10)

---

## 2. Desktop Core and lifecycle

### 2.1 Bootstrap and lifecycle
- [x] **CORE-01** (P0) Deterministic startup order: services declare `DependsOn()` and are sorted topologically; cycles give an error. ✅ Test order and cycle detection. ⛓ FIX-09
- [x] **CORE-02** (P0) Service contract in `internal/core/registry`:
  ```go
  type Service interface {
      Name() string
      Start(ctx context.Context) error
      Stop(ctx context.Context) error
      Health(ctx context.Context) contracts.ServiceInfo
  }
  ```
  `Reloader` is optional; services are `required` or `optional` (optional ⟵ degraded). ✅
- [x] **CORE-03** (P0) Staged rollback: a required service fails ⟵ stop all services already started; an optional one fails ⟵ `degraded` and continue. ✅ Test with a fake service.
- [x] **CORE-04** (P0) Ordered shutdown with a timeout per service (not only a global one); a stuck service is killed and logged. ✅ Test a hang.
- [x] **CORE-05** (P0) Signal tests: SIGTERM/SIGINT/SIGHUP (= reload) and clear exit codes (extend the existing e2e `t_daemon_lifecycle`).
- [x] **CORE-06** (P1) Single instance: clear error message plus the PID of the old process; detect stale locks. ⛓ FIX-08
- [x] **CORE-07** (P1) Daemon status and the reason for the last failure are kept in state (`status` API) and in `state_dir/last-failure.json`.

### 2.2 Process supervisor (`internal/app/supervisor`)
- [x] **CORE-10** (P0) Run processes with `exec.CommandContext` and an argument list (no shell); validate the path with `LookPath` and check owner/permissions. ✅
- [x] **CORE-11** (P0) Capture stdout/stderr into structured logs (line-split, size-capped); record exit code/signal. ✅
- [x] **CORE-12** (P0) Restart policy: `max_restarts` within a `window`, exponential backoff with jitter; above the limit ⟵ `failed/degraded`, no loop. ✅ Test with a process that always crashes.
- [x] **CORE-13** (P0) Stopping a child: SIGTERM ⟵ grace period ⟵ SIGKILL on the process group (`Setpgid`); no orphans after the daemon dies (`Pdeathsig`). ✅ e2e test.
- [x] **CORE-14** (P1) Environment allowlist for children, and redact sensitive variables (`*TOKEN*`, `*SECRET*`, `*PASSWORD*`) in logs.

### 2.3 Event bus and state
- [x] **CORE-20** (P0) `internal/core/eventbus` with generics: `Subscribe[T](topic) (<-chan T, cancel)`. ✅
- [x] **CORE-21** (P0) Documented and tested backpressure policy: bounded queue; coalesce state events (only the latest value matters); never silently drop essential events (report an error); isolate handler panics.
- [x] **CORE-22** (P0) Define the main events in `internal/api/events` with a versioned schema: `SessionStarted`, `SwayConnected/Disconnected`, `WorkspaceChanged`, `OutputChanged`, `ConfigChanged`, `ServiceStateChanged`, `ShutdownRequested`.
- [x] **CORE-23** (P0) `internal/core/state`: immutable snapshot plus a monotonic version (for snapshot recovery in the API); `-race` test and a benchmark. ⛓ API-09
- [x] **CORE-24** (P1) Load test (10k events/s, 100 subscribers) and goroutine leak check with `goleak`.

---

## 3. Configuration (`internal/core/config`)

> Today only `paths.go` exists, so everything below is open.

- [x] **CFG-01** (P0) Typed Go structs for the four files plus `version`, and a strict parser (`yaml.v3` with `KnownFields(true)`); record the new dependencies. ✅ Valid/invalid fixtures.
- [x] **CFG-02** (P0) A written schema (JSON Schema or a Go validator) and error messages with key path and line number (`desktop.yaml:12 ui.theme: unknown value "x"`).
- [x] **CFG-03** (P0) `//go:embed` for `configs/defaults/*` and fill in valid values (`shortcuts.yaml` / `plugins.yaml` must not be just empty lists: add real default shortcuts at least).
- [ ] **CFG-04** (P0) Documented and built layer order: defaults < `/etc/i3-next` < `~/.config/i3-next` < env (`I3NEXT_*`) < flags. ✅ Merge test.
- [ ] **CFG-05** (P0) Install defaults on first run with `O_EXCL` (never overwrite). ✅ Idempotency test.
- [ ] **CFG-06** (P0) Atomic writes: temp file in the same directory ⟵ `fsync` ⟵ `rename` ⟵ `fsync(dir)`; set `0600/0644` permissions. ✅ Crash-injection test.
- [ ] **CFG-07** (P0) Invalid config: the daemon starts with the last valid config (or defaults), state is `degraded` with an actionable error; no restart loop. ✅ e2e.
- [ ] **CFG-08** (P1) Versioned migration (`vN ⟶ vN+1`) with an automatic backup first, and a golden-file test.
- [ ] **CFG-09** (P1) Safe reload (SIGHUP + `config.reload` API): compute a diff; report non-reloadable keys clearly.
- [ ] **CFG-10** (P1) `i3-nextctl config validate|show|backup|restore`. ⛓ OBS-05
- [ ] **CFG-11** (P1) `docs/config/` full key reference (generated from the schema) plus examples.
- [ ] **CFG-12** (P1) Optional file watcher (`fsnotify`) with debounce for reload.

---

## 4. Sway integration (`internal/adapters/sway`)

### 4.1 IPC client
- [ ] **SWAY-01** (P0) Standalone client: find `SWAYSOCK` (fallback to `swaymsg -t get_socket_path` only if needed), clear error if missing. ✅
- [ ] **SWAY-02** (P0) i3-ipc protocol: magic `i3-ipc`, header (len, type) in native endianness; framing tested with fuzz (`go test -fuzz`) for partial/huge messages.
- [ ] **SWAY-03** (P0) Commands (`RUN_COMMAND`), queries (`GET_WORKSPACES/OUTPUTS/TREE/INPUTS/VERSION`), and subscribe (window/workspace/output/mode/shutdown/input).
- [ ] **SWAY-04** (P0) Reconnect with backoff + resubscribe + full snapshot rebuild after connecting; handle Sway `shutdown` events correctly. ✅ e2e: kill/restart Sway.
- [ ] **SWAY-05** (P0) Context/timeout for every request; safe concurrency (one reader goroutine, matching replies by message type).
- [ ] **SWAY-06** (P0) Typed models plus real JSON fixtures from Sway 1.8 and 1.10+ in `internal/adapters/sway/testdata/`. ✅ Parse test that tolerates unknown fields.
- [ ] **SWAY-07** (P1) An `Adapter` interface in `internal/adapters/sway` (services depend only on it) plus a fake for service tests.
- [ ] **SWAY-08** (P1) Version-based feature detection (a capability table in code and docs).
- [ ] **SWAY-09** (P0) Integration test with headless Sway in `make vm-test` (using the existing `tests/e2e` setup).

### 4.2 Window and workspace management
- [ ] **SWAY-20** (P0) `workspace` / `window-manager` service: snapshot + events ⟵ event bus. (⛓ CORE-20/22)
- [ ] **SWAY-21** (P0) Operations: focus/move/close/floating/layout/move-to-workspace/output. Inputs (workspace name, app_id) must be **escaped/validated** so command injection into Sway is impossible. ✅ Test with an input like `; exec ...`.
- [ ] **SWAY-22** (P1) Multi-monitor: workspace↔output, hotplug (connect the existing `sway_monitor_hotplug` test to the service).
- [ ] **SWAY-23** (P1) Window rules with a schema; generate `window-rules` from config instead of a fixed file.
- [ ] **SWAY-24** (P0) Generate the Sway config from a template (`text/template`) into `state_dir/generated/`, validate with `sway --validate -c`, and **never** overwrite the user's config without a backup/consent. ✅
- [ ] **SWAY-25** (P1) Document the default keybindings and how to return to the default config.

---

## 5. Core ⟷ Shell API (`internal/api/rpc`)

- [ ] **API-01** (P0) Recorded decision (ADR in `docs/architecture/adr/0001-rpc.md`): Unix socket + JSON-RPC 2.0 (line- or length-prefixed).
- [ ] **API-02** (P0) Socket at `$XDG_RUNTIME_DIR/i3-next/i3-next.sock` with `0600`; directory `0700`; set `umask` before `Listen`; remove a stale socket only if connecting fails. ✅ Permission test.
- [ ] **API-03** (P0) Check `SO_PEERCRED`: only the session owner's UID is allowed. ✅ Test that another UID is rejected (in the VM with a second user).
- [ ] **API-04** (P0) `hello {api_version, client}` handshake and a compatible/incompatible reply (`contracts.APIVersion`); show `version_mismatch` in the UI.
- [ ] **API-05** (P0) Limits: max message size (for example 1 MiB), per-request timeout, max concurrent requests, max connections. ✅ Fuzz/over-limit test.
- [ ] **API-06** (P0) Error mapping: `internal/core/errors` ⟷ `contracts.ErrorCode` in one table, with a test.
- [ ] **API-07** (P0) Minimum v1 methods: `status.get`, `service.list/restart`, `config.get/validate/reload`, `workspace.*`, `output.*`, `launcher.search/launch`, `notification.*`, `session.logout/restart/shutdown`.
- [ ] **API-08** (P0) Event stream: `events.subscribe` (notifications without id) with a sequence number.
- [ ] **API-09** (P0) Recovery after reconnect: `state.snapshot` with a `version`; the client detects a gap and fetches a new snapshot. ⛓ CORE-23
- [ ] **API-10** (P1) `docs/api/` docs (generated from code) plus a `socat` example.
- [ ] **API-11** (P1) Shared contract test: golden JSON files that both Go and QML (mock backend) test against.
- [ ] **API-12** (P1) Rate limit on sensitive methods (shutdown/launch).

---

## 6. Quickshell desktop shell

### 6.1 Foundation
- [ ] **UI-01** (P0) Real `Backend.qml`: connect to the socket (Quickshell `Socket` / `SplitParser`), handshake, reconnect with backoff, states `disconnected/connecting/connected/incompatible`. ⛓ API-04
- [ ] **UI-02** (P0) `EventBridge.qml`: parse events and fill QML models (`ListModel`/`QtObject`) without blocking the UI thread.
- [ ] **UI-03** (P0) A real startup flow `Main→Shell→Session` and a working `qmldir` (module `i3_next` with a discoverable import path, installed in `/usr/share/i3-next/quickshell`).
- [ ] **UI-04** (P0) `Variants`/`Scope` per monitor; hotplug works (panel is created/removed). ✅ e2e with `create_output` / `unplug`.
- [ ] **UI-05** (P0) Error handling: one failing component must not take down the whole shell (Loader with `onStatusChanged`); if the shell crashes, the supervisor restarts it. (⛓ CORE-12)
- [ ] **UI-06** (P1) Separate theme tokens (`themes/default/Tokens.qml`) and live change from `Settings.qml`.
- [ ] **UI-07** (P1) Logging/debug mode (`I3NEXT_UI_DEBUG=1`) and send errors to the daemon.
- [ ] **UI-08** (P1) Different scales/resolutions; accessibility (focus, keyboard navigation, contrast ≥ 4.5:1, minimum font size).
- [ ] **UI-09** (P0) `qmllint` required and clean in CI; smoke test: `quickshell -p ui/quickshell` starts with no errors in the VM.

### 6.2 Features (suggested order)
- [ ] **UI-10** (P0) Panel: clock, workspaces, network, audio, battery (hidden if there is no battery). ⛓ SVC-*
- [ ] **UI-11** (P0) Launcher: read desktop entries (XDG, `NoDisplay/Hidden/OnlyShowIn`), fuzzy search, safe launch (no shell; handle field codes `%u %f` correctly).
- [ ] **UI-12** (P0) Session menu: logout/restart/shutdown with confirmation.
- [ ] **UI-13** (P1) Notification popup + Notification Center. (⛓ SVC-03)
- [ ] **UI-14** (P1) Control Center (audio, brightness, network, Bluetooth if present).
- [ ] **UI-15** (P1) OSD (volume/brightness/workspace).
- [ ] **UI-16** (P1) Task/window list.
- [ ] **UI-17** (P1) Empty/loading/error/disconnected states for every view (checklist in the PR template).
- [ ] **UI-18** (P1) Basic appearance/theme settings.

---

## 7. System services

### 7.1 Shared rules (before any service)
- [ ] **SVC-00** (P0) Shared service template (`internal/services/<name>`): `enabled` config, optional dependency ⟵ `degraded` with a reason, timeout/cancel on operations, publish `ServiceStateChanged`, tests with a fake adapter. ✅ "Missing dependency" test.

### 7.2 Must-have
- [ ] **SVC-01** (P0) Session manager. (⛓ SES-*)
- [ ] **SVC-02** (P1) Wallpaper: `swaybg` under the supervisor, multi-monitor.
- [ ] **SVC-03** (P1) Notifications: implement `org.freedesktop.Notifications` on the session bus (clear error if another daemon owns the name), support actions/urgency/timeout, limit count and size. ⛓ DBUS-01
- [ ] **SVC-04** (P1) Audio: PipeWire/WirePlumber (`wpctl` with an argument list, or a library), volume/mute/default device, change events.
- [ ] **SVC-05** (P1) Network: NetworkManager D-Bus (read-only status in v1; no NM ⟵ degraded).
- [ ] **SVC-06** (P1) Power: UPower (battery/AC) + logind (suspend/reboot/poweroff) with polkit.
- [ ] **SVC-07** (P0) Lock screen: `swaylock` (a trusted tool); no fake lock. If the component crashes it must **fail closed** (the session must not open; with `ext-session-lock` the screen stays locked when the client dies). ✅ Manual/automatic test by killing swaylock.
- [ ] **SVC-08** (P1) Idle: `swayidle` with a policy from config (lock/DPMS/suspend) and inhibit.
- [ ] **SVC-09** (P1) Screenshot: `grim` / `slurp`, save to `XDG_PICTURES_DIR`, safe file names.
- [ ] **SVC-10** (P1) Clipboard: `wl-clipboard` (get/set within v1 scope, no persistent history).
- [ ] **SVC-11** (P1) Display: outputs/resolution/scale with mode validation (from `get_outputs`) and automatic revert after N seconds (like "Keep changes?").
- [ ] **SVC-12** (P1) Media: MPRIS over D-Bus.
- [ ] **SVC-13** (P1) Brightness (for Control Center): `brightnessctl` / logind `SetBrightness`.

### 7.3 Can wait (P2)
- [ ] Full Bluetooth, file search / command palette, clipboard history, layout manager, color picker, system monitor, workspace overview / animations, power profiles.

---

## 8. D-Bus and system integration (`internal/adapters/dbus`)

- [ ] **DBUS-01** (P0/P1) Choose a library (`godbus/dbus/v5`) + wrapper with reconnect/timeout and a split between session and system bus. A "which service uses which bus" table in `docs/architecture/dbus.md`.
- [ ] **DBUS-02** (P1) Fake bus for tests (or `dbus-run-session` in e2e).
- [ ] **DBUS-03** (P1) Validate D-Bus inputs (size/type/UTF-8).
- [ ] **DBUS-04** (P1) polkit/logind for power: least privilege, no `sudo`.
- [ ] **DBUS-05** (P2) D-Bus service file for activation (if needed). (⛓ PKG-05)
- [ ] **DBUS-06** (P1) Test in an incomplete session (no UPower/NM/…).

---

## 9. Plugin system

- [ ] **PLG-00** (P0) **Final v1 decision:** suggested = external plugins **disabled** in v1; built-ins only. Record in an ADR and in `RELEASE_SCOPE.md`. ✅ No third-party loading code in the v1 build (or put it behind a `plugins` build tag).
- [ ] **PLG-01** (P2) If enabled: versioned manifest + schema, permission model, trust/signing, resource limits, sandbox (bubblewrap/landlock/seccomp) with the real security boundary documented honestly.
- [ ] **PLG-02** (P2) SDK and sample plugin, tests for malicious/broken plugins, API compatibility tests.
- [ ] **PLG-03** (P1) If `plugins.yaml` is filled in v1: unknown plugin = rejected with a clear message.

---

## 10. Storage

- [ ] **STO-00** (P0) Decide which data really needs to persist. (Suggestion for v1: only small state, maybe even without SQLite, using atomic JSON.) If SQLite is not needed, remove it from the README/architecture or move it to v1.x.
- [ ] **STO-01** (P1) If needed: pure-Go SQLite (`modernc.org/sqlite`, no CGO), WAL, `0600`, versioned migrations (`internal/storage/migrations`), repository interface.
- [ ] **STO-02** (P1) DB corruption / failed migration: automatic backup, `quarantine`, start with a fresh DB and a warning; tests.
- [ ] **STO-03** (P1) Retention/cleanup policy for cache and history; do not store sensitive data.
- [ ] **STO-04** (P1) Backup/restore: `i3-nextctl backup|restore` (config + state).

---

## 11. Security and privacy

- [ ] **SEC-01** (P0) `docs/SECURITY.md` (threat model): assets, trust boundaries (socket, D-Bus, config, child processes, plugins), short STRIDE, assumptions.
- [ ] **SEC-02** (P0) Review permissions of all files/sockets/locks with automatic tests. (⛓ FIX-07, API-02)
- [ ] **SEC-03** (P0) Path traversal / symlinks: open files with `os.Root` / `openat2(RESOLVE_BENEATH)` or check `EvalSymlinks` + owner; for config, assets, and desktop entries.
- [ ] **SEC-04** (P0) Security lint: `gosec` is on; no pointless `//nolint`. Add `govulncheck`. (⛓ FIX-12)
- [ ] **SEC-05** (P0) Central input validation for IPC/D-Bus/config; fuzz the parsers.
- [ ] **SEC-06** (P0) No secrets or personal paths in logs (redaction helper + test).
- [ ] **SEC-07** (P0) Lock fails closed. (⛓ SVC-07)
- [ ] **SEC-08** (P1) Harden the systemd user unit (`NoNewPrivileges`, `ProtectSystem`, … as far as the session allows).
- [ ] **SEC-09** (P1) Supply chain: pin actions/tools, `go mod verify`, SBOM, release signing (cosign/minisign), reproducible builds (`-trimpath` is already used).
- [ ] **SEC-10** (P1) `THIRD_PARTY_LICENSES.md` (generated with `go-licenses`) and a license compatibility check.
- [ ] **SEC-11** (P0) Telemetry policy: **none** (documented in README/PRIVACY.md).
- [ ] **SEC-12** (P1) Vulnerability disclosure process: the email / private reporting must really work (the current `SECURITY.md` is vague).

---

## 12. Logging, diagnostics, observability

- [ ] **OBS-01** (P0) Logger with fixed fields `component`, `version`, `commit`, `session_id`; output to journald in a real session and to `state_dir/logs/` in dev mode; rotation/size cap (`lumberjack`, or rely on journald).
- [ ] **OBS-02** (P1) Log level from config and at runtime (`i3-nextctl log-level`); `AddSource` only in debug (already done).
- [ ] **OBS-03** (P1) Actionable errors (a `hint=` key) separate from normal errors.
- [ ] **OBS-04** (P0) Rewrite `i3-next-doctor`: checks use the shape `{id,status,detail,remedy}`; cover Sway (+version), Quickshell (+version), Qt6, Wayland env, D-Bus session/system, optional tools (swaylock, grim, wpctl, …), directory permissions, valid config, API socket; `--json`; exit codes (0 OK, 1 WARN with `--strict`, 2 FAIL). ✅ e2e `t_doctor_*`.
- [ ] **OBS-05** (P0) `i3-nextctl`: a real API client; commands `status`, `stop`, `restart`, `config validate`, `services`, `log-level`; `--json` output; standard exit codes; remove the current `status` that prints a fixed string. ✅ e2e: with the daemon running, `status` ⟵ `running`.
- [ ] **OBS-06** (P1) `i3-nextctl diagnostics bundle` with paths/user name/IP redacted.
- [ ] **OBS-07** (P2) Simple internal metrics (counter/gauge in `status`) with no network export.

---

## 13. Session, login, and session management

- [ ] **SES-01** (P0) `packaging/wayland-sessions/i3-next.desktop` (`Exec=i3-next-session`, `Type=Application`).
- [ ] **SES-02** (P0) Real `i3-next-session`: set env (`XDG_CURRENT_DESKTOP=i3-next:sway`, `XDG_SESSION_TYPE=wayland`, `XDG_SESSION_DESKTOP`, `MOZ_ENABLE_WAYLAND`, `QT_QPA_PLATFORM`…), run `dbus-update-activation-environment --systemd`, `systemctl --user import-environment`, start Sway, then `i3-nextd` / Quickshell in a fixed order. Using `systemd --user` targets (`i3-next-session.target`) is recommended. ✅
- [ ] **SES-03** (P0) Prevent duplicate starts (lock / one `systemctl --user` unit).
- [ ] **SES-04** (P0) Startup failure: show a message, log to `state_dir/logs/session-*.log`, fall back to a TTY/foot, and document recovery. ✅ Test with a broken daemon.
- [ ] **SES-05** (P0) Clean logout: stop services (reverse order), `swaymsg exit`, remove socket/lock.
- [ ] **SES-06** (P1) Restart the shell without logging out (`i3-nextctl shell restart`).
- [ ] **SES-07** (P1) Suspend/resume and lock-before-sleep (logind inhibitor); manual test in QA.
- [ ] **SES-08** (P1) Document dev mode (nested Sway) and real mode (display manager).
- [ ] **SES-09** (P0) VM test: `make vm-gui` brings up a full session (autologin) with no manual steps, and a screenshot is saved.

---

## 14. Packaging and install

### 14.1 Build / release
- [ ] **PKG-01** (P0) `make build` is the only build source; remove `scripts/build.sh` or make it a thin wrapper; standard `make install DESTDIR=… PREFIX=…` (binaries, configs, QML, session, docs).
- [ ] **PKG-02** (P0) Same ldflags from `internal/version`. (⛓ FIX-04) ✅ In a release, `i3-nextd --version` shows the real commit and date (not `dev/unknown`).
- [ ] **PKG-03** (P1) Separate debug/release builds; `-trimpath`, `CGO_ENABLED=0` for release.
- [ ] **PKG-04** (P1) SBOM (`syft` / `cyclonedx-gomod`), checksums, signing.

### 14.2 Packages
- [ ] **PKG-10** (P0) `.deb` for Ubuntu 24.04 and Debian 13 (`packaging/debian/`): `Depends` (sway, quickshell\*, qt6 runtime, dbus-user-session, …), `Recommends` (pipewire, network-manager, upower, swaylock, swayidle, grim, slurp, wl-clipboard), clean `lintian`. ✅ Installs on a clean VM of both distros. (\*Quickshell may be missing on Ubuntu 24.04 ⟵ decide: PPA / backport / vendoring, and write it in the docs.)
- [ ] **PKG-11** (P0) postinst/prerm: never touch user config; upgrade keeps config; purge only with consent.
- [ ] **PKG-12** (P1) systemd user units and dbus/polkit files.
- [ ] **PKG-13** (P2) PKGBUILD (Arch, best effort).
- [ ] **PKG-14** (P1) Automatic install/upgrade/uninstall test on a VM. (⛓ QA-05)
- [ ] **PKG-15** (P1) Decide and document: install from a repository (APT repo / release assets) or offline.

### 14.3 Scripts
- [ ] **PKG-20** (P1) `scripts/dev-setup.sh` is idempotent; `scripts/install.sh` has `--prefix`, `--dry-run`, checks prerequisites, and touches **only files it owns** (install manifest); `uninstall.sh` uses the same manifest. ✅ shellcheck + test with a path that has spaces.
- [ ] **PKG-21** (P1) No script overwrites a user file without a backup.

---

## 15. Testing and QA

### 15.1 Go
- [ ] **QA-01** (P0) Unit tests for every new package (target: critical paths in core/config/api/sway ≥ 80% line coverage; focus on behavior).
- [ ] **QA-02** (P0) `go test -race -count=1 ./...` + `goleak`.
- [ ] **QA-03** (P1) Fuzz: IPC framing, JSON-RPC, config parser, desktop entry parser.
- [ ] **QA-04** (P1) Fix flaky tests (`-count=20` in nightly).

### 15.2 Integration / E2E (building on `tests/e2e/run.sh`)
- [ ] **QA-05** (P0) New e2e tests: daemon+Sway connect/reconnect, real `nextctl status`, socket permissions, broken config, full display-manager login (VM GUI), logout, crash of daemon/shell/lock, multi-monitor/scale, missing optional deps, upgrade/uninstall, `.deb` install.
- [ ] **QA-06** (P1) Run e2e in CI (a runner with `/dev/kvm`, or a headless-only subset without a VM) + upload artifacts (`tests/vm/.state/artifacts`).
- [ ] **QA-07** (P1) Long soak test (≥ 8h) measuring RSS/FD/goroutines.
- [ ] **QA-08** (P1) VM matrix: Debian 13 and Ubuntu 24.04 (a second image in `vm.sh`).

### 15.3 Manual QA
- [ ] **QA-10** (P0) `docs/development/qa-checklist.md`: keyboard/mouse/touchpad/multi-monitor, sleep/resume, network connect/disconnect, audio device change, GTK/Qt/Electron apps, many notifications, recovery from broken config/daemon/UI.
- [ ] **QA-11** (P0) At least one full day-to-day use cycle on real hardware (result in `docs/release/rc-report.md`).

---

## 16. Performance and reliability

- [ ] **PERF-01** (P1) Baseline and budget (in `docs/release/performance.md`): login→desktop ready ≤ 5s, idle daemon RAM ≤ 50 MiB, shell ≤ 250 MiB, idle CPU < 2%.
- [ ] **PERF-02** (P1) Measure startup time of each component (trace timings in the log).
- [ ] **PERF-03** (P1) Debounce/throttle frequent events (`window::title`, `input`) without losing essential events.
- [ ] **PERF-04** (P1) Memory limits for queues/caches. (⛓ CORE-21)
- [ ] **PERF-05** (P1) Tests for disk full, EACCES, process hang, full socket.
- [ ] **PERF-06** (P1) `go test -bench` benchmarks for event bus/state/IPC parser, with a baseline in CI (warn on regression).

---

## 17. Documentation

- [ ] **DOC-01** (P0) README: real status, install commands per distro (Ubuntu 24.04 / Debian 13), full dependency list (including `foot`, `jq`, `rsync`) matching `guest-provision.sh`, build/run/test, link to the KVM lab.
- [ ] **DOC-02** (P0) `docs/user-guide/`: install, choosing the session in the display manager, first run, shortcuts, config.
- [ ] **DOC-03** (P0) `docs/troubleshooting.md` + `docs/recovery.md` (broken config, daemon will not start, black UI, going back to plain Sway).
- [ ] **DOC-04** (P1) `docs/api/` and `docs/config/`. (⛓ API-10, CFG-11)
- [ ] **DOC-05** (P1) Backup/restore/upgrade/uninstall.
- [ ] **DOC-06** (P1) ADRs (`docs/architecture/adr/`) for the RPC, plugin, storage, and session decisions.
- [ ] **DOC-07** (P1) Update `OVERVIEW.md` with a real diagram and the "Core does not depend on UI" rule.
- [ ] **DOC-08** (P1) Real UI screenshots (from `vm.sh screenshot`) and Known Issues.
- [ ] **DOC-09** (P2) Plugin guide, only if plugins are enabled.

---

## 18. CI/CD and release engineering

- [ ] **CI-01** (P0) Strengthen the current CI (⛓ FIX-12): matrix, cache, pinning, `govulncheck`, real `qmllint`, shellcheck.
- [ ] **CI-02** (P1) Build job on Ubuntu 24.04 and Debian 13 (containers) and build the `.deb` as an artifact.
- [ ] **CI-03** (P1) e2e/VM job (self-hosted or nightly). (⛓ QA-06)
- [ ] **CI-04** (P0) Release workflow on tag `vX.Y.Z`: build, test, `.deb`, checksums, SBOM, signing, GitHub Release with changelog.
- [ ] **CI-05** (P1) Dependabot/Renovate for Go modules and Actions.
- [ ] **CI-06** (P1) Conventional Commits check or automatic changelog.
- [ ] **CI-07** (P0) A release candidate (`v1.0.0-rc.N`) before the final; document rollback / installing the previous version.
- [ ] **CI-08** (P1) Branch protection: PRs required, CI green, at least one review.

---

## 19. Release Gate — final checklist

> All items must be true at the same time. Link each item to an automatic test or recorded QA.

- [ ] `.deb` installs on a clean Ubuntu 24.04 **and** Debian 13 system (QA-05 / PKG-14)
- [ ] Login from the display manager into the i3_next session; Sway + daemon + Quickshell start automatically (SES-09)
- [ ] Panel and launcher are usable (UI-10/11)
- [ ] Window/workspace management and multi-monitor work (SWAY-20..22)
- [ ] Notifications work (SVC-03 / UI-13)
- [ ] Audio/network/battery are reported correctly (SVC-04..06)
- [ ] Lock/logout/reboot/shutdown are safe; lock fails closed (SVC-07, SES-05)
- [ ] If the backend disconnects the UI does not die and it reconnects (UI-01, API-09)
- [ ] Broken config does not lose the session or cause a restart loop (CFG-07)
- [ ] Uninstall/upgrade do not delete user data (PKG-11)
- [ ] CI is green; no open blocker/critical bugs
- [ ] `govulncheck` and dependency scan show no critical findings (SEC-04)
- [ ] Install/troubleshooting/recovery docs are published (DOC-01..03)
- [ ] Version/commit/date/changelog are visible in artifacts (PKG-02)
- [ ] RC tested on real hardware and the result is recorded (QA-11)
- [ ] Known Issues and the bug-report path are published (DOC-08, SEC-12)

---

## 20. Milestones with exit criteria

| # | Milestone | Sections | Exit criteria (measurable) |
|---|---|---|---|
| M0 | Scope & quick wins | 0, 1 (`FIX-*`) | All P0 `FIX-*` closed; `make`, `make check`, `make vm-test` green; one version everywhere |
| M1 | Core runtime | 2, 3, 12 | Daemon with fake services: order/rollback/shutdown/supervisor/config tested; real `nextctl status` |
| M2 | Sway & API | 4, 5 | e2e: daemon ⟷ headless Sway, reconnect, JSON-RPC with version + peercred, contract tests |
| M3 | First usable shell | 6, 7 (Must) | Panel + launcher + session menu on the VM; multi-monitor; UI reconnect |
| M4 | Session | 8, 13 | Login from the DM in the VM with no manual steps; clean logout; recovery documented |
| M5 | Persistence & security | 10, 11 | Threat model; permissions; fuzz; lock fails closed; clean `govulncheck` |
| M6 | Packaging & docs | 14, 17 | `.deb` install/upgrade/uninstall on both distros; docs complete |
| M7 | QA & hardening | 15, 16 | 8-hour soak with no leaks; performance budget; manual QA |
| M8 | RC & release | 18, 19 | `v1.0.0-rc.N` ⟵ full Release Gate ⟵ `v1.0.0` |

### Known risks (track these)
| Risk | Impact | Mitigation |
|---|---|---|
| Quickshell is missing or old on Ubuntu 24.04 / Debian 13 | Tier-1 install blocker | Decide early: PPA / backport / vendoring; test in the VM in M3 |
| Go `1.27.3` may not be available on distros/CI | Build blocker | FIX-05 in M0 |
| Insecure or fake lock screen | Security risk | Only swaylock / ext-session-lock + fail-closed test |
| UI/service scope is large | Delays | Re-prioritize P1/P2 in M3; Must services are only section 7.2 P0/P1 |
| No KVM runner in CI | e2e only runs locally | Headless subset without a VM in CI + self-hosted nightly |

---

## Release progress log

| Item | Value |
|---|---|
| Target version | `1.0.0` |
| Current status | Planning → M0 (quick wins) |
| Release owner | TBD |
| Target distributions | Ubuntu LTS 24.04, Debian Stable 13 (amd64) |
| Target release date | TBD |
| Latest release candidate | TBD |
| Known blockers | FIX-05 (Go version), Quickshell on Tier-1, decisions on PLG-00 / STO-00 |
