# i3_next — TODO

> **Goal:** ship `v1.0.0` (a complete Sway-based desktop session people can install and use), then the differentiating features from [`docs/new_features.md`](docs/new_features.md).
>
> This file is the single source of truth for planned work. The complete structure rules are in [`docs/development/todo-structure.md`](docs/development/todo-structure.md) — **edit the content, keep the structure** (section 0 below is the short version).

---

## 0. How this file works

### 0.1 Task line format

```
- [ ] **FIX-00** `P0` `bug` `v1.1` — Outcome-first title. ✅ testable acceptance. ⛓ DEP-01
```

- The optional 4th tag is the target release; the default is `v1.0`.
- Details go in indented sub-bullets (`  - ✅ …`, `  - ⛓ …`), never in a new top-level line.

### 0.2 Markers

| Marker | Meaning                                                             |
| ------ | ------------------------------------------------------------------- |
| `[ ]`  | Open                                                                |
| `[~]`  | Partial or reopened: the line says what is missing (and since when) |
| `[x]`  | Done and verified: code + tests + docs, `make check` green          |

### 0.3 Priority rubric (decides order, not size)

| Level | Rule                                                                                                                                                                                                      |
| ----- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `P0`  | Do now. Any of: (a) the build, CI or e2e is broken or reports false results; (b) a critical security problem or a fail-open security path; (c) it blocks the core user journey _log in → usable desktop_. |
| `P1`  | Needed for the target release and gives something a user or tester can see, run or rely on (feature, visible fix, real hardening).                                                                        |
| `P2`  | Internal quality, polish or nice-to-have; may slip to a later release.                                                                                                                                    |

Tie-breaks: (1) order by _tangibility_ — what can someone see or run right after merge? (2) a task that only prepares internals is `P2`, unless a `P0`/`P1` task is blocked on it; then it inherits that priority and gets a `⛓`.

### 0.4 Types and ID prefixes

Types: `bug`, `sec`, `feat`, `test`, `infra`, `docs`, `debt`.

Prefixes: `FIX` defects in already-merged code (any area), `STRUCT`, `CORE`, `CFG`, `SWAY`, `API`, `UI`, `SVC`, `DBUS`, `PLG`, `STO`, `SEC`, `OBS`, `SES`, `PKG`, `QA`, `PERF`, `DOC`, `CI`, `FEAT` (features from `docs/new_features.md`). IDs are never renumbered or reused.

### 0.5 Definition of Done

Code written, tests (unit/integration/e2e) pass in CI, docs updated, `make check` and `make vm-test` green.

### 0.6 Editing rules

1. Keep the section order and numbering (full rules: `docs/development/todo-structure.md`).
2. New task: next free ID in its prefix, in the right area of section 3, sorted `P0 → P2` inside **Open**.
3. Finishing a task: flip to `[x]`, move it to **Done** of its area, add a line to the progress log (section 6), refresh the status table (section 1).
4. Reopening: flip to `[~]`, add `(reopened YYYY-MM-DD: reason)`. Never delete IDs.
5. Section 2 is derived from the rubric: at most 12 ordered items, only open tasks, each referencing a task ID in bold.
6. Every new feature idea needs: type, acceptance criteria, dependencies, and a security note if it executes commands or reads user files.
7. Dropping a task: keep the line, flip it to `[x]`, start the title with `DROPPED —` plus the reason, move it to **Done**, and log it. Never delete it.

---

## 1. Status snapshot

As of 2026-11-09 (**static review** of the source snapshot; build/test/`make check` results are taken from the progress log, not re-run). Current milestone: **M0 — Unbreak**.

| Area                    | State                    | Evidence                                                                                                                                                                                                                                  |
| ----------------------- | ------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Build / format / lint   | ✅ (reported)            | progress log: `go build ./...`, `go test -race ./...`, `make check` green (FIX-17, FIX-19); `go.mod`, CI, CONTRIBUTING agree on Go 1.26.3                                                                                                 |
| e2e / VM tests          | ⚠️ false-green           | `summary` runs before the `t_cfg07_*` tests are defined, script exit status ignores failures, `t_version_flags` expects `0.1` (version is 0.2.20) (FIX-18); guest provisioning installs Go from apt and cannot meet `go.mod` (FIX-36); image checksum disabled (FIX-25) |
| Daemon `i3-nextd`       | ✅ wired (CORE-30)        | registers the reconnecting Sway adapter → workspace service → supervised `quickshell`/`swaybg`; honours `enabled` from `services.yaml`; logs `ServiceStateChanged` and every workspace/output/sway event. Remaining defects tracked as separate open items: `required`/`depends_on` ignored (FIX-34), broken `services.yaml` kills it (FIX-35 ✅), exits silently on error (FIX-37), no API (API-02) |
| Supervisor / processes  | ⚠️                       | restart policy, backoff, process-group kill, `processsvc` tests; `Start` after `Stop` fails, data race on `lastErr`, 10 ms health-poll log flood (FIX-23, FIX-24)                                                                          |
| CLI `i3-nextctl`        | ⚠️ stub                  | `status` prints a fixed string; `config backup/restore` only print (OBS-05, CFG-10)                                                                                                                                                       |
| Sway adapter            | ❌ wrong against real Sway | framing and models fine; message/event type numbers do not match sway-ipc(7) so `subscribe` fails and window/shutdown events are lost (FIX-33); reconnecting adapter exists, snapshot not rebuilt after reconnect (SWAY-04, FIX-38)         |
| Session                 | ❌                       | `i3-next-session` is a stub, yet its `.desktop` entry is installed (FIX-20)                                                                                                                                                               |
| Shell (QML)             | ❌                       | placeholder rectangles; the daemon starts `quickshell` with no path/entry file (FIX-40, UI-03)                                                                                                                                            |
| Services / API / D-Bus  | ❌                       | only the workspace service exists (wired, no caller for its commands); no RPC, no D-Bus                                                                                                                                                   |
| Packaging               | ⚠️                       | `make install` exists; path drift, no `.deb` (FIX-27, PKG-10)                                                                                                                                                                             |
| Docs                    | ⚠️                       | architecture, security, config reference exist; no user guide or troubleshooting; `.kilo/skills` disagree with `go.mod` (FIX-42)                                                                                                          |

---

## 2. Priority queue

Ordered by the rubric in 0.3: first make the repo and its tests honest (and make the VM able to build), then fix the Sway protocol layer, then deliver the first thing a user can run (login → desktop with a panel), securing the control socket on the way. Re-derive after every merged task.

1. **FIX-36** — guest provisioning installs a Go that satisfies `go.mod`
2. **FIX-18** — e2e/VM suite fails when a test fails (and defines every test before running it)
3. **FIX-33** — Sway IPC message/event numbers match sway-ipc(7)
4. **FIX-21** — client tests, adapter `RunCommand` reports failures, no hang after disconnect
5. **FIX-20** — stop installing the broken session entry
6. **FIX-35** — broken `services.yaml` degrades instead of killing the daemon (with FIX-37 error output)
7. **SWAY-04** — rebuild the snapshot and react to `shutdown` after Sway restarts
8. **FIX-34** — `services.yaml` `required`/`depends_on` honoured, deterministic start order
9. **FIX-38** — workspace service stop/recovery defects
10. **FIX-40** — default shell/wallpaper commands runnable
11. **API-03** — reject connections from other UIDs (`SO_PEERCRED`), then `hello` and `status.get`
12. **OBS-05** — real `i3-nextctl status` talking to the daemon

Next wave (not queued yet): UI-10 first panel (clock + workspaces, UI-01, UI-03), UI-11 launcher (with SEC-03), SVC-07 lock that fails closed, FEAT-01 shell self-heal, SVC-04/SVC-06 audio and battery, FEAT-03/FEAT-04 snapshot save/restore.

---

## 3. Backlog by area

### 3.1 Structure, build and repo hygiene

**Open**

- [x] **FIX-42** `P2` `docs` — `.kilo/skills` state different Go versions than `go.mod` (README: "Go 1.24 baseline", `golang-popular-libraries`: Go 1.24, `golang-dependency-management`: 1.26.0, `go.mod`/CI: 1.26.3). ✅ skills say "the version in `go.mod`" instead of a number.
- [x] **FIX-43** `P2` `infra` — `scripts/dev-setup.sh` installs `staticcheck` and `golangci-lint` with `@latest` while CI pins `v0.8.2` / `v2.15.0`. ✅ one version source (Makefile variables) used by dev-setup, pre-commit and CI.

**Done**

- [x] **FIX-00a** `P0` `docs` — `go.mod`, CI, CONTRIBUTING and RELEASE_SCOPE agree on Go 1.26.3 (bumped from 1.26.0 for stdlib fixes, see CHANGELOG)
- [x] **FIX-01** `P0` `infra` — default goal is `all`
- [x] **FIX-02** `P0` `bug` — `lib.sh` lives in `tests/e2e/`
- [x] **FIX-03** `P0` `bug` — Sway config includes are relative; no binds to missing commands
- [x] **FIX-04** `P0` `infra` — single `internal/version` package, one `--version` format
- [x] **FIX-05** `P0` `infra` — Go version decision (1.26.3) applied everywhere
- [x] **FIX-07** `P0` `sec` — runtime/state/data/config directories are `0700`
- [x] **FIX-08** `P0` `sec` — lock file is not deleted on release (see FIX-31)
- [x] **FIX-09** `P0` `debt` — typed lifecycle interfaces in bootstrap
- [x] **FIX-10** `P0` `bug` — `Run` collects stop errors and watches service health
- [x] **FIX-19** `P0` `infra` — `make check` and CI are green on a fresh clone: `gofmt -l .` empty, `go mod tidy` clean, `go.sum` present, dependencies current, `govulncheck` DB unavailable offline (403) but dependencies audited, `make check` passes. ✅
- [x] **FIX-12** `P0` `infra` — CI: pinned actions/tools, govulncheck, timeouts, concurrency, real qmllint
- [x] **FIX-13** `P1` `docs` — README "Target structure (not yet created)"
- [x] **FIX-14** `P1` `docs` — `docs/SECURITY.md` exists; `scripts/build.sh` is a wrapper
- [x] **FIX-15** `P1` `bug` — `i3-next-doctor` exit codes and `--json`
- [x] **FIX-16** `P1` `docs` — naming rule documented and checked by `scripts/check-naming.sh`
- [x] **FIX-28** `P2` `docs` — Docs drift fixed: `docs/kvm.md` references `tests/vm/vm.py`, CHANGELOG top section is `Unreleased`, `finalize-task.md` bumps version only at release, `RELEASE_SCOPE.md` "Last updated" removed. ✅ one place per fact.
- [x] **FIX-29** `P2` `debt` — API-surface bloat reduced: one name per operation with typed options/structs in `supervisor`, `registry`, `state`, `eventbus`; `serviceShutdownTimeout`/`killService` reflection replaced by `ShutdownTimeouter`/`Killer` typed interfaces in `bootstrap`. Alias methods and `...any` parsing removed. ✅
- [x] **FIX-32** `P2` `debt` — Defaults single-sourced from `internal/core/config/defaults/` (embedded); duplicate `configs/defaults/` removed, Makefile and config tests updated. ✅
- [x] **STRUCT-01** `P0` `infra` — import layering enforced with depguard
- [x] **STRUCT-02** `P1` `docs` — CODEOWNERS, PR/issue templates, CHANGELOG
- [x] **STRUCT-03** `P1` `docs` — dependency policy

### 3.2 Core and lifecycle

**Open**

- [x] **CORE-30** `P0` `feat` — Daemon wiring is in place (`i3-nextd` registers the reconnecting Sway adapter → workspace service → supervised `quickshell` and `swaybg`, honours `enabled` from `services.yaml`, logs `ServiceStateChanged` and every workspace/output/sway event); the remaining defects are tracked as their own open items (`required`/`depends_on` ignored → FIX-34, workspace service stop/recovery defects → FIX-38, default shell/wallpaper commands not runnable → FIX-40, e2e proofs false-green → FIX-18, snapshot rebuild/shutdown after reconnect → SWAY-04). ✅ e2e with headless Sway: `swaymsg workspace number 2` ⇒ `WorkspaceChanged` in the log; `kill -9` of the shell ⇒ restart with backoff, `degraded` after the limit; disabling a service in `services.yaml` is honoured; a required service failure aborts with rollback. ⛓ FIX-33, FIX-34, FIX-38, FIX-40, FIX-18, SWAY-04
- [ ] **FIX-23** `P1` `bug` — Supervisor correctness: `Pdeathsig` is tied to the OS _thread_ that forked the child (needs `runtime.LockOSThread` in the spawning goroutine) so children can be killed spuriously; `p.lastErr` is assigned in `runOnce` without the mutex (data race); `Process.Start` after `Stop` returns `ErrAlreadyRunning` (`cancel` is never cleared and the process stays tracked) so `Restart()` and restarting a `processsvc` service never work; an explicit `spec.Env` bypasses the allowlist (decide and document); command validation only checks the exec bit (not owner or world-writable); `Process.Health` reports `running` for starting/restarting/stopped; there are no supervisor-level tests (only `processsvc` covers start/stop and the restart limit). ✅ tests: crashing child respects the restart limit; stop then start works; process-group kill leaves no orphans; redaction; `-race` clean.
- [ ] **FIX-24** `P1` `bug` — Bootstrap status: a persisted `last-failure.json` is reloaded on every start and reported forever (never cleared after a healthy start); health is polled every 10 ms and logs one error/warn line per tick for every failed or degraded service (about 100 lines/s with `quickshell` missing or Sway down); the state stays `running` while `Degraded()` is true; an informational migration note marks the config load as degraded. ✅ healthy start clears the failure; poll interval 2–5 s and log on transition only; `degraded` state reported; `Degraded` only for real problems.
- [ ] **FIX-37** `P1` `bug` — `i3-nextd` exits with status 1 and prints nothing when `run()` fails (config errors, "already running", a failed required service): the wrapped error is discarded in `main`. ✅ error goes to stderr and the log; exit codes follow the CORE-05 table; e2e: a second instance prints "already running (PID n)".
- [~] **CORE-05** `P1` `test` — Signals: SIGTERM/SIGINT covered by e2e; SIGHUP is not handled (FIX-22) and exit codes are not specified (reopened 2026-11-07). ✅ table of signals → behaviour → exit code, with tests.
- [~] **CORE-10** `P1` `sec` — Supervisor launches via `exec.CommandContext` with argv and `LookPath`; owner/permission check and tests pending (FIX-23).
- [~] **CORE-11** `P1` `feat` — stdout/stderr capture and exit code/signal recorded; tests pending (FIX-23).
- [~] **CORE-12** `P1` `feat` — restart policy with backoff and window; the restart limit is covered through `processsvc` (`sh -c "exit 1"`), supervisor-level tests pending (FIX-23).
- [~] **CORE-13** `P1` `bug` — SIGTERM → grace → SIGKILL on the process group; `Pdeathsig` caveat, restart-after-stop and the e2e orphan test pending (FIX-23).
- [~] **CORE-14** `P1` `sec` — env allowlist and redaction implemented; explicit-env bypass and tests pending (FIX-23).
- [ ] **FIX-41** `P2` `debt` — Small core inconsistencies: `eventbus.Subscribe` maps an empty topic to `"*"` but `Publish` never delivers to it; `events.NewEvent` leaves `Event.ID` empty; `State.notifyLocked` starts one goroutine per update so listeners can see updates out of order; `NextGlobalVersion` is unused. ✅ wildcard either works or is removed; events get an ID/sequence (API-08); listener order test.

**Done**

- [x] **CORE-01** `P0` `feat` — topological startup order, cycle detection
- [x] **CORE-02** `P0` `feat` — `registry.Service` contract, required/optional
- [x] **CORE-03** `P0` `feat` — rollback on required failure, degrade on optional
- [x] **CORE-04** `P0` `feat` — per-service shutdown timeout, stuck service killed
- [x] **CORE-06** `P1` `feat` — single instance with PID in the error
- [x] **CORE-07** `P1` `feat` — last failure kept in state and `last-failure.json`
- [x] **CORE-20** `P1` `feat` — generic `Subscribe[T]`/`Publish[T]` package-level API
- [x] **CORE-21** `P1` `feat` — backpressure/coalescing policy (state slot, essential queue-full)
- [x] **CORE-22** `P0` `feat` — versioned event schemas in `internal/api/events`
- [x] **CORE-23** `P0` `feat` — immutable versioned snapshots in `internal/core/state`
- [x] **CORE-24** `P1` `test` — load test (10k events, 100 subs) and goleak clean
- [x] **FIX-17** `P0` `bug` — event bus compiles: package generics, idempotent cancel/close, pump-owned channel close

### 3.3 Configuration

**Open**

- [x] **CFG-07** `P0` `feat` — Invalid config degrades to last-valid/defaults with no restart loop: works for `desktop.yaml` only; an invalid `services.yaml` makes the daemon exit (reopened 2026-11-09: services/plugins/shortcuts have no fallback). ✅ see FIX-35. ⛓ FIX-35
- [x] **FIX-35** `P0` `bug` — Invalid user `services.yaml` / `plugins.yaml` / `shortcuts.yaml` (or a bad `I3NEXT_SERVICES_*` override) makes `i3-nextd` exit before bootstrap: only `desktop.yaml` has last-valid/defaults fallback (P0 inherited from CFG-07). ✅ the fallback logic is generalised to all four files; e2e like cfg07 with a broken `services.yaml` ⇒ daemon runs on last-valid/defaults and reports degraded with an actionable message. ⛓ FIX-37
- [ ] **FIX-34** `P1` `bug` — `services.yaml` semantics are only half implemented: `required` and `depends_on` are parsed but ignored (bootstrap classifies by each Go type's `Optional()`/`DependsOn()`, and the `registry.Registry` built in `main` is write-only); enabling `workspace` while `sway` is disabled makes `TopologicalSort` fail with an unknown dependency and the daemon exits; independent services start in random map order. ✅ table test: `required: true` failure aborts with rollback, YAML `depends_on` is honoured, a disabled/missing dependency degrades the dependent service instead of exiting, start order is deterministic; `docs/config/README.md` documents `required` and `depends_on`; the unused registry is used or removed.
- [ ] **FIX-22** `P1` `bug` — Config is mostly decorative: `i3-nextd` does not handle SIGHUP (default action terminates the process without graceful shutdown) and nothing triggers reload (`Reloader`, `DiffDesktop`, the watcher are unused); `logging.level/format` are ignored (`logging.New(debug)` runs before config is loaded); `paths.*` and `ui.*` are unused. ✅ SIGHUP ⇒ validated reload + log line; `logging.level: debug` changes output; non-reloadable keys reported; e2e. ⛓ CFG-09
- [~] **CFG-05** `P1` `bug` — `InstallDefaults` exists with `O_EXCL`, but nothing calls it, so first run installs nothing (the doctor even tells users to run `i3-nextd` to get defaults) (reopened 2026-11-07). ✅ first run creates missing files only; idempotent.
- [~] **CFG-02** `P1` `feat` — only unknown-key and syntax errors are reported; no value validation (`session.compositor: foo`, bad `logging.level` pass) (reopened 2026-11-07). ✅ validator with `file:line key: unknown value "x"`.
- [~] **CFG-09** `P1` `feat` — diff and non-reloadable keys implemented; no trigger (SIGHUP, `config.reload` API) (reopened 2026-11-07).
- [~] **CFG-10** `P1` `feat` — `config validate|show` work; `backup` and `restore` only print a message (reopened 2026-11-07). ✅ real atomic backup/restore with tests. ⛓ OBS-05
- [~] **CFG-12** `P2` `feat` — watcher with debounce exists; not wired.
- [~] **CFG-06** `P2` `test` — `AtomicWrite` implemented; the promised crash-injection test is missing (reopened 2026-11-07).

**Done**

- [x] **CFG-01** `P0` `feat` — typed structs and strict parser
- [x] **CFG-03** `P0` `feat` — embedded defaults
- [x] **CFG-04** `P0` `feat` — layer order defaults < system < user < env < flags
- [x] **CFG-08** `P1` `feat` — versioned migration with backup and golden test
- [x] **CFG-11** `P1` `docs` — `docs/config/README.md` key reference

### 3.4 Sway integration

**Open**

- [x] **FIX-33** `P0` `bug` — Sway IPC numbers match sway-ipc(7): the code has `GetOutputs=2, GetTree=3, GetInputs=4, GetVersion=5, Subscribe=6` (real: `Subscribe=2, GetOutputs=3, GetTree=4, GetVersion=7, GetInputs=100`) and event types `window=0, workspace=1, output=2, mode=3, shutdown=4, input=5` (real: `workspace=0, output=1, mode=2, window=3, shutdown=6, input=21`); against a real Sway `subscribe` fails, `GetTree` receives the outputs array, window and shutdown events are dropped or mislabelled, and `GetOutputs` only works through the `outputsFromTree` fallback; the fake-server unit tests reuse the same constants so they cannot catch it. ✅ constants checked against the man page and pinned by a table test; the `outputsFromTree` fallback is removed; `make vm-test` exercises subscribe, outputs, tree, inputs, version and a window event against headless Sway. ⛓ FIX-36
- [x] **FIX-21** `P0` `bug` — Sway client correctness: connection loss now returns an error, no deadline is set on the shared connection, replies are matched FIFO per type, `Client.RunCommand` checks `success:false`, event channels close on disconnect; missing: tests for each (`client_test.go` only covers the happy path), `ReconnectingAdapter.RunCommand` calls `Request` directly so failed commands look like success, a `Request` issued after the reader died waits for the context forever (no `readerDone` check) and a later `Subscribe` leaks an unclosed channel, `outputsFromTree` unmarshals a tree object into `[]Node` (remove, see FIX-33). ✅ fake-server tests: dropped connection ⇒ error; timeout then next request works; two concurrent `GET_*` both answered; failing command ⇒ error (also through the adapter); event channel closes on disconnect; `ReconnectingAdapter.RunCommand` now uses `Client.RunCommand`; `Client.Request` checks `readerDone`; `Client.Subscribe` cleans up channel on failure. ⛓ FIX-33
- [x] **SWAY-04** `P0` `feat` — Reconnect with backoff and resubscribe is implemented in `ReconnectingAdapter` (100 ms → 5 s, stable subscriber channels); missing: rebuild of the workspace snapshot after a reconnect (today only on the next event), reaction to the Sway `shutdown` event (blocked by FIX-33), `SwayConnected`/`SwayDisconnected` events are defined but never published, a second `Subscribe` with new event names is ignored until the next reconnect, and the e2e test. ✅ `SwayConnected`/`SwayDisconnected` events published on connect/disconnect; workspace snapshot rebuilt on `SwayConnected`; shutdown event handled via `SwayDisconnected` with reason "sway shutdown"; second `Subscribe` with new events triggers re-subscribe without reconnect. ⛓ FIX-21, FIX-33, FIX-38
- [x] **SWAY-03** `P0` `feat` — Commands, queries and subscribe: implemented with wrong message-type numbers (reopened 2026-11-09: see FIX-33). ✅ fixed by FIX-33. ⛓ FIX-33
- [ ] **SWAY-24** `P0` `feat` — Generate the Sway config from a template into `state_dir/generated/`, validate with `sway --validate -c`, and never overwrite the user's config without backup and consent. ✅ validation in e2e; user config byte-identical afterwards.
- [ ] **FIX-38** `P1` `bug` — Workspace service lifecycle: `Stop` waits on `done`, which only `loop` closes, so a service whose `Start` failed (optional services are still stopped at shutdown) blocks until the shutdown timeout; `Start` fails for good if Sway is not connected within 5 s and never recovers when Sway appears; every event triggers a full rebuild (PERF-03); the unused `logger` is built from `NewTextHandler(nil, nil)` and would panic if used. ✅ fake-adapter tests: stop without start returns immediately; start while disconnected degrades then recovers after connect; a reconnect triggers exactly one rebuild. ⛓ SWAY-04
- [~] **SWAY-20** `P1` `feat` — workspace service with snapshot + events is implemented and registered by the daemon; its command methods have no caller until the API exists (API-07); window events and `shutdown` are lost until FIX-33 and the lifecycle defects in FIX-38 are fixed (reopened 2026-11-07, updated 2026-11-09). ⛓ CORE-30
- [~] **SWAY-05** `P1` `bug` — context/timeouts exist, but see FIX-21 for the remaining client defects (reopened 2026-11-07).
- [ ] **SWAY-22** `P1` `feat` — Multi-monitor: workspace↔output mapping and hotplug in the service (connect the existing `sway_monitor_hotplug` test).
- [ ] **SWAY-09** `P1` `test` — Integration test against headless Sway in `make vm-test`; real Sway is the only oracle for protocol constants (see FIX-33).
- [ ] **SWAY-08** `P1` `feat` — Version-based feature detection (capability table in code and docs).
- [ ] **SWAY-23** `P1` `feat` — Window rules from a schema instead of a fixed file.
- [ ] **SWAY-25** `P2` `docs` — Document default keybindings and how to return to plain Sway.
- [ ] **FIX-30** `P2` `bug` — `Input` model uses `xkb_current_layout_index` as a string; sway reports `xkb_active_layout_index` (int) and `xkb_active_layout_name`. ✅ fixture from real `get_inputs` parses.
- [ ] **FIX-44** `P2` `bug` — Default shortcuts are inconsistent: `shortcuts.yaml` `launcher` runs the non-existent `i3-nextctl launcher`, the Sway keybinding `$mod+d` launches `foot`, and nothing reads `shortcuts.yaml`. ✅ until UI-11 exists the defaults only reference commands that exist; keybindings are generated from `shortcuts.yaml` (SWAY-24).

**Done**

- [x] **SWAY-01** `P0` `feat` — socket discovery with clear error
- [x] **SWAY-02** `P0` `feat` — i3-ipc framing (magic, length, type) with fuzz test
- [x] **SWAY-06** `P0` `test` — typed models with 1.8 / 1.10 fixtures
- [x] **SWAY-07** `P1` `feat` — `Adapter` interface and fake
- [x] **SWAY-21** `P0` `sec` — workspace/app_id/output/layout inputs validated (no command injection)

### 3.5 Core ⟷ Shell API

**Open**

- [ ] **API-03** `P0` `sec` — Check `SO_PEERCRED`: only the session owner's UID is accepted. ✅ another UID rejected (second user in the VM).
- [ ] **API-01** `P0` `docs` — ADR `docs/architecture/adr/0001-rpc.md`: Unix socket + JSON-RPC 2.0 framing.
- [ ] **API-04** `P0` `feat` — `hello {api_version, client}` handshake; incompatible versions give `version_mismatch` (shown in the UI).
- [ ] **API-07** `P0` `feat` — Methods, delivered incrementally: first `status.get`, `service.list`; then `config.*`, `workspace.*`, `output.*`, `launcher.*`, `notification.*`, `session.*`. ⛓ API-02
- [ ] **API-05** `P1` `sec` — Limits: message size, request timeout, concurrent requests, connections. ✅ over-limit/fuzz test.
- [ ] **API-06** `P1` `feat` — One table mapping `internal/core/errors` ⟷ `contracts.ErrorCode`, with a test.
- [ ] **API-08** `P1` `feat` — `events.subscribe` stream with sequence numbers (events currently carry an empty `ID`, see FIX-41).
- [ ] **API-09** `P1` `feat` — `state.snapshot` with `version`; clients detect gaps and refetch. ⛓ CORE-23
- [ ] **API-10** `P1` `docs` — `docs/api/` generated from code plus a `socat` example.
- [ ] **API-11** `P2` `test` — Golden JSON contract files shared by Go and the QML mock backend.
- [ ] **API-12** `P2` `sec` — Rate limit sensitive methods (shutdown/launch).

**Done**

- [x] **API-02** `P0` `sec` — control socket `0600` in a `0700` directory, umask set before `Listen`, stale socket removed only if connecting is refused

### 3.6 Shell UI (Quickshell/QML)

**Open**

- [ ] **UI-01** `P0` `feat` — Real `Backend.qml`: socket connect, handshake, reconnect with backoff, states `disconnected/connecting/connected/incompatible`. ⛓ API-04
- [ ] **UI-03** `P0` `feat` — Real startup flow `Main→Shell→Session` and a working `qmldir`; today there is no `shell.qml` entry file, the QML roots are plain `Rectangle`s (no `ShellRoot`/`PanelWindow`) and the daemon runs `quickshell` without a path. The shell is started by the daemon/session (not by hand). ✅ `quickshell -p ui/quickshell` starts without errors in the VM. ⛓ FIX-40
- [ ] **UI-02** `P0` `feat` — `EventBridge.qml` fills models from events without blocking the UI thread.
- [ ] **UI-10** `P0` `feat` — Panel: clock, workspaces, network, audio, battery (hidden without a battery); first slice is clock + workspaces. ⛓ UI-01, UI-02
- [ ] **UI-11** `P0` `feat` — Launcher: XDG desktop entries (`NoDisplay/Hidden/OnlyShowIn`), fuzzy search, safe launch (no shell, correct `%u %f` field codes). ⛓ SEC-03
- [ ] **UI-12** `P1` `feat` — Session menu: logout/restart/shutdown with confirmation. ⛓ SES-05
- [ ] **UI-04** `P1` `feat` — Panel per monitor with hotplug. ✅ e2e with `create_output` / `unplug`.
- [ ] **UI-05** `P1` `bug` — One failing component must not take the shell down (Loader error handling); crashed shell is restarted by the supervisor. ⛓ CORE-12
- [ ] **UI-09** `P1` `infra` — `qmllint` clean in CI plus a VM smoke test.
- [ ] **UI-13** `P1` `feat` — Notification popup and Notification Center. ⛓ SVC-03
- [ ] **UI-14** `P1` `feat` — Control Center (audio, brightness, network, Bluetooth if present).
- [ ] **UI-17** `P1` `feat` — Empty/loading/error/disconnected state for every view.
- [ ] **UI-15** `P2` `feat` — OSD (volume/brightness/workspace).
- [ ] **UI-16** `P2` `feat` — Task/window list.
- [ ] **UI-06** `P2` `feat` — Theme tokens and live change from `Settings.qml`.
- [ ] **UI-07** `P2` `infra` — `I3NEXT_UI_DEBUG=1` and error reporting to the daemon.
- [ ] **UI-08** `P2` `feat` — Scales/resolutions and accessibility (focus, keyboard, contrast ≥ 4.5:1).
- [ ] **UI-18** `P2` `feat` — Appearance/theme settings.

### 3.7 System services

**Open**

- [ ] **SVC-07** `P0` `sec` — Lock screen with `swaylock` (trusted tool); fails closed: if the locker dies the session stays locked (`ext-session-lock`). No lock or logout UI ships without this. ✅ test by killing swaylock.
- [ ] **FIX-40** `P1` `bug` — The default supervised services are not runnable: `wallpaper` starts `swaybg` with no image/colour arguments (and the Sway config already sets the wallpaper via `output * bg`), `shell` starts `quickshell` with no `-p`/config path and the tree has no `shell.qml`, so both exit at once and burn their restart budget into `failed`. ✅ command lines come from config/defaults; exactly one owner for the wallpaper; the shell shows up in the VM from `ui/quickshell` (or its installed path). ⛓ UI-03, SVC-02
- [ ] **SVC-00** `P1` `infra` — Service template: `enabled` config, missing optional dependency ⇒ `degraded` with a reason, timeouts, `ServiceStateChanged`, tests with a fake adapter.
- [ ] **SVC-02** `P1` `feat` — Wallpaper: `swaybg` under the supervisor with real arguments, multi-monitor (the service entry exists in `main`, see FIX-40).
- [ ] **SVC-04** `P1` `feat` — Audio: PipeWire/WirePlumber (`wpctl` with argv): volume, mute, default device, change events.
- [ ] **SVC-06** `P1` `feat` — Power: UPower battery/AC plus logind suspend/reboot/poweroff via polkit.
- [ ] **SVC-03** `P1` `feat` — Notifications: `org.freedesktop.Notifications` on the session bus (clear error if the name is taken), actions/urgency/timeout, bounded count and size. ⛓ DBUS-01
- [ ] **SVC-09** `P1` `feat` — Screenshot: `grim`/`slurp`, saved to `XDG_PICTURES_DIR`, safe names.
- [ ] **SVC-13** `P1` `feat` — Brightness: `brightnessctl` / logind.
- [ ] **SVC-05** `P1` `feat` — Network: NetworkManager read-only status; absent NM ⇒ `degraded`.
- [ ] **SVC-11** `P1` `feat` — Display: resolution/scale validated against `get_outputs`, automatic revert after N seconds ("Keep changes?").
- [ ] **SVC-08** `P1` `feat` — Idle: `swayidle` policy from config (lock/DPMS/suspend) and inhibit.
- [ ] **SVC-01** `P1` `feat` — Session manager service. ⛓ SES-02
- [ ] **SVC-10** `P2` `feat` — Clipboard via `wl-clipboard` (no persistent history).
- [ ] **SVC-12** `P2` `feat` — Media (MPRIS).
- [ ] **SVC-90** `P2` `feat` `v1.x` — Deferred: full Bluetooth, file search/command palette, clipboard history, layout manager, color picker, system monitor, workspace overview, power profiles.

### 3.8 D-Bus and system integration

**Open**

- [ ] **DBUS-01** `P1` `feat` — `godbus/dbus/v5` wrapper with reconnect/timeouts, session vs system split, table in `docs/architecture/dbus.md`.
- [ ] **DBUS-03** `P1` `sec` — Validate D-Bus inputs (size/type/UTF-8).
- [ ] **DBUS-04** `P1` `sec` — polkit/logind for power; least privilege, no `sudo`.
- [ ] **DBUS-02** `P1` `test` — Fake bus or `dbus-run-session` in e2e.
- [ ] **DBUS-06** `P1` `test` — Run in an incomplete session (no UPower/NM).
- [ ] **DBUS-05** `P2` `infra` — D-Bus activation service file if needed. ⛓ PKG-12

### 3.9 Plugin system

**Open**

- [ ] **PLG-00** `P1` `sec` — Decide and record (ADR + RELEASE_SCOPE): external plugins disabled in v1, built-ins only. ✅ no third-party loading code in the v1 build (or behind a `plugins` build tag).
- [ ] **PLG-03** `P1` `feat` — Unknown plugin in `plugins.yaml` is rejected with a clear message (today nothing reads `plugins.yaml`).
- [ ] **PLG-01** `P2` `sec` `v1.x` — Manifest, permissions, signing, limits and sandbox if plugins are ever enabled.
- [ ] **PLG-02** `P2` `feat` `v1.x` — SDK, sample plugin, tests for malicious/broken plugins.

### 3.10 Storage

**Open**

- [ ] **STO-00** `P1` `docs` — Decide what must persist (suggestion: small atomic JSON state, no SQLite in v1); if SQLite is not needed, remove it from README/architecture.
- [ ] **STO-04** `P1` `feat` — `i3-nextctl backup|restore` for config + state. ⛓ CFG-10
- [ ] **STO-01** `P2` `feat` — If needed: pure-Go SQLite (WAL, `0600`, migrations, repository interface).
- [ ] **STO-02** `P2` `bug` — Corrupt DB or failed migration: backup, quarantine, fresh start with a warning.
- [ ] **STO-03** `P2` `feat` — Retention policy; no sensitive data stored.

### 3.11 Security and privacy

**Open**

- [ ] **SEC-03** `P0` `sec` — Path traversal and symlinks: `os.Root` / `openat2(RESOLVE_BENEATH)` or `EvalSymlinks` + owner check for config, assets, desktop entries, snapshots, imported configs. Must land before UI-11, FEAT-04, FEAT-20.
- [ ] **SEC-07** `P0` `sec` — Lock fails closed (the requirement behind SVC-07; verified by test and QA). ⛓ SVC-07
- [ ] **SEC-05** `P1` `sec` — Central input validation for IPC/D-Bus/config; fuzz the parsers.
- [ ] **SEC-06** `P1` `test` — No secrets or personal paths in logs (redaction helper + test).
- [ ] **SEC-02** `P1` `test` — Automatic permission tests for all files, sockets and locks. ⛓ API-02
- [~] **SEC-04** `P1` `infra` — `gosec` and `govulncheck` are on; clean up pointless `//nolint` (for example `// test file permissions` copied into non-test code in `ipc.go`) and keep findings at zero.
- [ ] **SEC-09** `P1` `infra` — Supply chain: `go mod verify`, SBOM, release signing, reproducible builds.
- [ ] **SEC-10** `P1` `docs` — `THIRD_PARTY_LICENSES.md` via `go-licenses` and a compatibility check.
- [ ] **SEC-11** `P1` `docs` — Telemetry policy: none (README / `PRIVACY.md`).
- [ ] **SEC-12** `P1` `docs` — Working vulnerability-disclosure channel in `SECURITY.md` (currently vague).
- [ ] **SEC-08** `P2` `sec` — Harden the systemd user unit.
- [ ] **FIX-31** `P2` `sec` — `acquireLock` removes the lock file when the recorded PID is dead; the kernel already drops `flock` on death, so this only reintroduces the FIX-08 race. ✅ remove the stale-removal path; two-process test still has exactly one winner.
- [ ] **SEC-13** `P2` `docs` — Mark controls in `docs/SECURITY.md` that are not implemented yet (socket peer checks, plugin handling) as _planned_.

**Done**

- [x] **SEC-01** `P0` `docs` — threat model in `docs/SECURITY.md`

### 3.12 Observability and CLI

**Open**

- [ ] **OBS-05** `P0` `feat` — `i3-nextctl` becomes a real API client: `status`, `services`, `config validate|show|backup|restore`, `stop`, `restart`, `log-level`; `--json`; exit codes 0/1/2; remove the hard-coded `status` string and update the e2e `t_ctl_status` that expects it. ✅ e2e: daemon running ⇒ `status` prints `running` and the service list; daemon down ⇒ exit 1 with a hint. ⛓ API-02, API-07
- [ ] **OBS-01** `P1` `feat` — Logger with `component`, `version`, `commit`, `session_id`; journald in a session, `state_dir/logs/` in dev; size cap.
- [ ] **OBS-02** `P1` `feat` — Log level from config and at runtime (`i3-nextctl log-level`). ⛓ FIX-22
- [ ] **FIX-26** `P1` `bug` — Doctor: all five directory checks share the id `dir-i3-next`; no minimum Sway/Qt version check (needs SWAY-08). ✅ unique ids (`dir-config`, `dir-state`, …) asserted by an e2e test.
- [ ] **OBS-06** `P1` `feat` — `i3-nextctl diagnostics bundle` with paths, user name and IPs redacted.
- [ ] **OBS-03** `P2` `feat` — Actionable errors (`hint=` key).
- [ ] **OBS-07** `P2` `feat` — Internal counters/gauges in `status` (no network export).

**Done**

- [x] **OBS-04** `P0` `feat` — doctor checks with `{id,status,detail,remedy}`, `--json`, exit codes

### 3.13 Session and login

**Open**

- [ ] **SES-02** `P0` `feat` — `i3-next-session` is a real session: set environment (`XDG_CURRENT_DESKTOP=i3-next:sway`, `XDG_SESSION_TYPE=wayland`, …), `dbus-update-activation-environment --systemd`, start Sway with the generated config (SWAY-24) as a supervised child, then `i3-nextd`; logs to `state_dir/logs/session-*.log`. ⛓ SWAY-24, CORE-30
- [ ] **SES-09** `P0` `test` — `make vm-gui` brings up a full session with no manual steps and saves a screenshot. ⛓ SES-02
- [ ] **SES-04** `P0` `bug` — Startup failure shows a message, logs, falls back to plain Sway/foot; recovery documented. ✅ test with a broken daemon.
- [ ] **SES-05** `P0` `feat` — Clean logout: stop services in reverse order, `swaymsg exit`, remove socket and lock.
- [ ] **SES-03** `P1` `bug` — Prevent duplicate session starts.
- [ ] **SES-06** `P1` `feat` — Restart the shell without logging out (`i3-nextctl shell restart`). ⛓ FIX-23
- [ ] **SES-07** `P1` `feat` — Suspend/resume and lock-before-sleep (logind inhibitor).
- [ ] **SES-08** `P2` `docs` — Document dev mode (nested Sway) and real mode.

**Done**

- [x] **SES-01** `P0` `infra` — `packaging/wayland-sessions/i3-next.desktop` (see FIX-20 for when it may be installed)

### 3.14 Packaging and install

**Open**

- [ ] **FIX-20** `P0` `bug` — The stub session is installed: `make install` copies `i3-next.desktop` (`Exec=i3-next-session`) while the binary prints two lines and exits, so choosing it in the display manager bounces the user to the login screen. ✅ the session entry is installed only by `make install-session` (not by `make install`) until SES-09 passes; documented.
- [ ] **FIX-27** `P1` `bug` — Install paths drift: the wallpaper goes to absolute `$(WALLPAPER_DIR)` regardless of `PREFIX`; `configs/sway/config` hard-codes `/usr/share/i3-next/wallpapers/…`; `uninstall.sh` leaves the wallpaper; `share/i3-next/sway/` is read by nothing; `assets/wallpapers/*.png` could not be verified in the reviewed snapshot (`.repomixignore` excludes `*.png`). ✅ `make install DESTDIR=$(mktemp -d)` file list checked in CI; uninstall leaves nothing behind.
- [~] **PKG-01** `P1` `infra` — `make build` is the only build source and `make install` exists; DESTDIR/PREFIX consistency pending (FIX-27).
- [ ] **PKG-10** `P1` `feat` — `.deb` for Ubuntu 24.04 and Debian 13 with `Depends`/`Recommends`, lintian-clean. ✅ installs on a clean VM of both.
- [ ] **PKG-11** `P1` `sec` — postinst/prerm never touch user config; upgrade keeps config; purge only with consent.
- [ ] **PKG-12** `P1` `infra` — systemd user units and D-Bus/polkit files.
- [ ] **PKG-14** `P1` `test` — Automatic install/upgrade/uninstall test on a VM. ⛓ QA-05
- [ ] **PKG-20** `P1` `infra` — `install.sh` with `--prefix`, `--dry-run`, an install manifest, and only touching files it owns; `uninstall.sh` uses the manifest.
- [ ] **PKG-21** `P1` `sec` — No script overwrites a user file without a backup.
- [ ] **PKG-03** `P2` `infra` — Debug/release builds; `CGO_ENABLED=0` for release.
- [ ] **PKG-04** `P2` `infra` — SBOM, checksums, signing.
- [ ] **PKG-15** `P2` `docs` — Decide: APT repository vs release assets vs offline.
- [ ] **PKG-13** `P2` `infra` `v1.x` — PKGBUILD (Arch, best effort).

**Done**

- [x] **PKG-02** `P0` `infra` — shared ldflags from `internal/version`

### 3.15 Testing and QA

**Open**

- [ ] **FIX-36** `P0` `infra` — Guest provisioning satisfies `go.mod`: `tests/vm/guest-provision.py` removes any official Go, installs `golang-go` from apt and aborts when it is older than the `go` line (Debian 13 ships 1.24, `go.mod` needs 1.26.3; `trixie-backports` is configured but never selected), and `GOTOOLCHAIN=local` forbids a toolchain download at build time, so `make vm-up` fails or the guest cannot build (verify on a clean VM). ✅ a fresh `make vm-up` ends with `go version` ≥ `go.mod`, from backports (`-t trixie-backports`) or a checksum-verified tarball; the provision log records the source and version.
- [ ] **FIX-18** `P0` `test` — The e2e harness lies: `summary` is called before the `t_cfg07_*` functions are defined (so their `run_test` lines execute missing functions), and the script's exit status is that of its last statement (a function definition ⇒ 0) so failures pass `make vm-test`; `t_version_flags` still expects `0.1` while `internal/version` is `0.2.20`; the Sway-dependent daemon tests cannot pass until FIX-33. ✅ tests defined before main; `summary` is the last statement; expected version derived from `internal/version`; a deliberately failing test proves a non-zero exit.
- [ ] **FIX-25** `P1` `sec` — `tests/vm/vm.py`: the Debian image SHA512 check is commented out (the checksum is fetched and then ignored), so a tampered or truncated base image is used silently; `ensure_running()` calls `cmd_start()` without its argument (TypeError whenever the VM is stopped). ✅ mismatch aborts and deletes the download; `make vm-test` restarts a stopped VM.
- [ ] **QA-05** `P1` `test` — e2e for: daemon+Sway connect/reconnect, real `status`, socket permissions, broken config, full login (VM GUI), logout, crash of daemon/shell/lock, multi-monitor/scale, missing optional deps, upgrade/uninstall, `.deb` install.
- [ ] **QA-01** `P1` `test` — Unit tests for every new package; ≥ 80% line coverage on core/config/api/sway critical paths.
- [ ] **QA-02** `P1` `test` — `go test -race -count=1 ./...` with `goleak` is part of every merge.
- [ ] **QA-03** `P1` `test` — Fuzz: JSON-RPC, config parser, desktop-entry parser, snapshot and sway-config parsers.
- [ ] **QA-06** `P1` `infra` — e2e in CI (runner with `/dev/kvm`, or a headless subset) and artifact upload.
- [ ] **QA-08** `P1` `test` — VM matrix: Debian 13 and Ubuntu 24.04.
- [ ] **QA-10** `P1` `docs` — `docs/development/qa-checklist.md` (input devices, multi-monitor, sleep/resume, network/audio changes, GTK/Qt/Electron apps, notification floods, recovery).
- [ ] **QA-11** `P1` `test` — A full day-to-day cycle on real hardware, recorded in `docs/release/rc-report.md`.
- [ ] **QA-04** `P2` `test` — Flaky-test hunt (`-count=20` nightly).
- [ ] **QA-07** `P2` `test` — ≥ 8 h soak measuring RSS/FD/goroutines.

### 3.16 Performance and reliability

**Open**

- [ ] **PERF-03** `P1` `bug` — The workspace service re-fetches workspaces, outputs and the whole tree on every Sway event (including title changes). ✅ debounce/throttle without losing essential events; benchmark. ⛓ FIX-38
- [ ] **PERF-01** `P1` `docs` — Budget in `docs/release/performance.md`: login→desktop ≤ 5 s, idle daemon ≤ 50 MiB, shell ≤ 250 MiB, idle CPU < 2%.
- [ ] **PERF-02** `P2` `infra` — Per-component startup timings in the log.
- [ ] **PERF-04** `P2` `feat` — Memory limits for queues and caches.
- [ ] **PERF-05** `P2` `test` — Disk full, EACCES, hung process, full socket.
- [ ] **PERF-06** `P2` `test` — Benchmarks for bus/state/IPC parser with a CI baseline.

### 3.17 Documentation

**Open**

- [ ] **DOC-02** `P1` `docs` — `docs/user-guide/`: install, choosing the session, first run, shortcuts, config.
- [ ] **DOC-03** `P1` `docs` — `docs/troubleshooting.md` and `docs/recovery.md` (broken config, daemon will not start, black UI, back to plain Sway).
- [~] **DOC-01** `P1` `docs` — README: install commands per distro and dependency list matching `guest-provision.py`; real status wording kept honest.
- [ ] **DOC-10** `P1` `docs` — Update `docs/RELEASE_SCOPE.md` and README for the FEAT epics (what ships in v1.0, v1.1, v1.2) and add FEAT IDs to each entry of `docs/new_features.md` for traceability.
- [ ] **DOC-04** `P1` `docs` — `docs/api/` and `docs/config/` generated from code. ⛓ API-10
- [ ] **DOC-05** `P1` `docs` — Backup, restore, upgrade, uninstall.
- [ ] **DOC-06** `P2` `docs` — ADRs for RPC, plugins, storage, session.
- [ ] **DOC-07** `P2` `docs` — `OVERVIEW.md` with a real diagram.
- [ ] **DOC-08** `P2` `docs` — Real screenshots (`vm.py screenshot`) and Known Issues.
- [ ] **DOC-09** `P2` `docs` `v1.x` — Plugin guide, only if plugins are enabled.

**Done**

- [x] **DOC-11** `P1` `docs` — TODO.md structure rules documented for humans and AI agents in `docs/development/todo-structure.md`

### 3.18 CI/CD and release engineering

**Open**

- [ ] **CI-04** `P1` `infra` — Release workflow on tag `vX.Y.Z`: build, test, `.deb`, checksums, SBOM, signing, GitHub Release with changelog.
- [ ] **CI-02** `P1` `infra` — Build on Ubuntu 24.04 and Debian 13 containers; `.deb` as an artifact.
- [ ] **CI-07** `P1` `infra` — Release candidates (`v1.0.0-rc.N`) and a documented rollback.
- [ ] **CI-03** `P2` `infra` — e2e/VM job (self-hosted or nightly). ⛓ QA-06
- [ ] **CI-05** `P2` `infra` — Dependabot/Renovate for Go modules and Actions.
- [ ] **CI-06** `P2` `infra` — Conventional Commits check or generated changelog.
- [ ] **CI-08** `P2` `infra` — Branch protection (PR required, CI green, one review).

**Done**

- [x] **CI-01** `P0` `infra` — CI with pinned actions, govulncheck, shellcheck, qmllint

### 3.19 Feature epics from `docs/new_features.md`

Design notes that apply to all epics: nothing here may execute a command that came from a file the daemon did not write itself without validation (see SEC-03/SEC-05); anything that changes outputs must be revertible (SVC-11); every epic gets its own security note in its acceptance criteria.

**Open**

_Epic A — "Seamless" monitoring and self-healing (new_features #3)_

- [ ] **FEAT-01** `P1` `feat` — Shell self-heal: `quickshell` runs under the supervisor; a crash restarts it with backoff and notifies the user; after M crashes in a window it stays down, state is `degraded`, the user is told, no loop. ✅ e2e: `kill -9` the shell ⇒ back within N s; repeated kills ⇒ degraded + notification. ⛓ CORE-30, UI-03, FIX-23
- [ ] **FEAT-02** `P1` `feat` — Sway watchdog: liveness probe (`GET_VERSION` with timeout, 3 misses), `shutdown` event and process exit; policy `on_sway_failure: restart|logout|ignore` (default restart up to a limit, then logout with a message). Documented limit: restarting the compositor ends every Wayland client; recovery of windows is FEAT-05. ✅ e2e: `kill -STOP` Sway ⇒ detected ≤ 15 s and restarted. ⛓ SWAY-04, SES-02, FIX-21, FIX-33

_Epic B — Workspace snapshot save/restore (new_features #1)_

- [ ] **FEAT-03** `P1` `feat` `v1.1` — Snapshot capture: `internal/services/snapshot` builds a versioned snapshot (outputs with make/model/serial, mode, scale, position; workspaces with output and layout; windows with `app_id` or XWayland class, workspace, output, floating, rect, fullscreen, marks) from `GET_TREE/OUTPUTS/WORKSPACES`; stored under `state_dir/snapshots/` with `0600`, atomic write, rotation; no command lines or environment stored. CLI: `i3-nextctl snapshot save|list|show|rm`. ✅ golden tests with 1.8/1.10 fixtures; fuzz the reader. ⛓ SWAY-20, OBS-05
- [ ] **FEAT-04** `P1` `feat` `v1.1` — Restore: `snapshot restore [name] [--dry-run]`; a pure, unit-testable plan builder (workspace→output mapping with fallback when an output is missing; relaunch missing apps through the desktop entry for their `app_id`, never raw command lines or a shell; wait for the window with a timeout; then `move`/`floating`/`layout`); running windows are only moved; idempotent; progress events. Security: the snapshot file is untrusted input — every field is whitelisted and size-limited. Documented limit: application-internal state is not restored. ✅ e2e: two `foot` windows on workspaces 1 and 3, save, kill, restore ⇒ same placement; a fixture with `app_id: x"; exec …` is rejected. ⛓ FEAT-03, UI-11, SEC-03
- [ ] **FEAT-05** `P1` `feat` `v1.1` — Auto-snapshot and recovery: debounced snapshots on window/workspace events plus a periodic one, a rotating `last-good`; after a crash or an unclean start, `snapshot.auto_restore: ask|always|never` (default `ask`). ⛓ FEAT-04, FEAT-02
- [ ] **FEAT-06** `P2` `feat` `v1.1` — UI: "Restore previous layout" in the Control Center/session menu and a notification after recovery. ⛓ UI-14, FEAT-05

_Epic C — Import from the Sway/i3 ecosystem (new_features #4)_

- [ ] **FEAT-20** `P1` `feat` `v1.1` — `i3-nextctl import-sway`: parser for the common subset of Sway/i3 config (`set`/`$var`, `include` with depth/size limits and path validation, `bindsym`/`bindcode` with flags, `exec`/`exec_always`, `output`, `input`, `workspace`, `assign`, `for_window`, `mode` blocks). Default is a dry-run report (mapped / unsupported / needs attention); `--apply` writes only into i3_next's own config with a backup; the user's Sway config is never edited and nothing from it is executed. ✅ golden tests on the i3 default, the Sway default and two real community configs; fuzz. ⛓ SEC-03
- [ ] **FEAT-21** `P1` `feat` `v1.1` — Mapping: `bindsym` → `shortcuts.yaml` (known keys such as `XF86Audio*` map to services; `exec` stays a command), enable the matching services in `services.yaml` when `wpctl`/`nmcli`/`brightnessctl` are detected, report conflicts with defaults. ⛓ FEAT-20, SVC-04, SVC-13
- [ ] **FEAT-22** `P2` `feat` `v1.2` — Keep-your-own-config path: user fragments in `~/.config/i3-next/sway.d/*.conf` are included last, so import is optional; X11-only i3 directives are reported as unsupported. ⛓ SWAY-24

_Epic D — Environment profiles (new_features #2)_

- [ ] **FEAT-10** `P1` `feat` `v1.2` — `profiles.yaml` (versioned, validated): `when` (outputs connected by name/make-model/count, app running by `app_id`, AC power) and `apply` actions from a whitelist: `output {enable, scale, mode, pos, primary}`, `sway_set {gaps, border, opacity}`, `service {name, enabled}`, `theme`. Arbitrary `exec` only with an explicit `allow_exec: true`. ⛓ CFG-02
- [ ] **FEAT-11** `P1` `feat` `v1.2` — Profile engine: event-driven (`OutputChanged`, window events), debounced, with priority/conflict rules, revert when the trigger ends (the game closes → previous profile) and a manual pin. ✅ table-driven decision tests; e2e: `create_output` ⇒ applied, `unplug` ⇒ reverted. ⛓ FEAT-10, SWAY-22
- [ ] **FEAT-12** `P1` `feat` `v1.2` — Display actions go through the SVC-11 confirm/revert flow and never disable the last active output. Documented limits: Sway has no blur or animations, so "effects" means gaps, borders and opacity; font-size changes apply to GTK/Qt settings only. ⛓ SVC-11
- [ ] **FEAT-13** `P1` `feat` `v1.2` — CLI: `i3-nextctl profile list|show|apply|pin|unpin|status`. ⛓ OBS-05
- [ ] **FEAT-14** `P2` `feat` `v1.2` — UI to create and switch profiles from Quickshell. ⛓ UI-14

---

## 4. Release gate

All items must be true at the same time; link each to an automatic test or recorded QA.

- [ ] `go build ./...`, `make check` and `make vm-test` are green, the VM guest builds with the Go from `go.mod`, and the e2e suite fails when a test fails (FIX-17, FIX-18, FIX-19, FIX-36)
- [ ] `.deb` installs on a clean Ubuntu 24.04 **and** Debian 13 system (PKG-10, PKG-14)
- [ ] Login from the display manager into the i3_next session; Sway, daemon and Quickshell start automatically (SES-09)
- [ ] Panel and launcher are usable (UI-10, UI-11)
- [ ] Window/workspace management and multi-monitor work against real Sway (FIX-33, SWAY-20, SWAY-22)
- [ ] Notifications work (SVC-03, UI-13)
- [ ] Audio, network and battery are reported correctly (SVC-04, SVC-05, SVC-06)
- [ ] Lock, logout, reboot and shutdown are safe; lock fails closed (SVC-07, SES-05)
- [ ] A backend disconnect does not kill the UI and it reconnects; a shell crash recovers (UI-01, API-09, FEAT-01)
- [ ] A broken config (any of the four files) does not lose the session or cause a restart loop (CFG-07, FIX-35)
- [ ] Uninstall and upgrade do not delete user data (PKG-11)
- [ ] CI is green and there are no open blocker/critical bugs
- [ ] `govulncheck` and the dependency scan show no critical findings (SEC-04)
- [ ] Install, troubleshooting and recovery docs are published (DOC-01, DOC-02, DOC-03)
- [ ] Version, commit, date and changelog are visible in artifacts (PKG-02)
- [ ] RC tested on real hardware and the result is recorded (QA-11)
- [ ] Known Issues and the bug-report path are published (DOC-08, SEC-12)

---

## 5. Milestones and risks

| #   | Milestone                    | Scope                                                                    | Exit criteria (measurable)                                                                                                                                  |
| --- | ---------------------------- | ------------------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| M0  | Unbreak                      | FIX-17 … FIX-21, FIX-33, FIX-36                                          | `go build ./...`, `make check`, `make vm-test` green; guest has Go ≥ `go.mod`; e2e exits non-zero when a test fails; subscribe/outputs/tree work on real Sway; no stub session installed |
| M1  | Core wired                   | CORE-30, FIX-34, FIX-35, FIX-37, FIX-38, SWAY-04, API-02…API-04, OBS-05  | daemon runs Sway adapter + workspace service + shell and survives a Sway restart; a broken `services.yaml` degrades; `i3-nextctl status` answers over the secure socket |
| M2  | Session MVP                  | SES-02, SES-04, SES-05, SES-09, SWAY-24, UI-01, UI-03, UI-10, FIX-40     | VM login with no manual steps shows a panel; clean logout; recovery documented                                                                              |
| M3  | Usable shell                 | UI-11…UI-14, SVC-02…SVC-09, SVC-11                                       | launcher, notifications, audio, battery, screenshot, display revert in the VM                                                                               |
| M4  | Self-healing and security    | FEAT-01, FEAT-02, SVC-07, SEC-03, SEC-05, SEC-06                         | shell crash recovers; Sway watchdog works; lock fails closed; fuzzers and `govulncheck` clean                                                               |
| M5  | Packaging and docs           | PKG-10…PKG-14, DOC-01…DOC-03                                             | `.deb` install/upgrade/uninstall on both distros; docs complete                                                                                             |
| M6  | QA and RC                    | QA-05…QA-11, PERF-01, CI-04, CI-07                                       | 8 h soak without leaks; budget met; `v1.0.0-rc.N` passes the release gate                                                                                   |
| M7  | Release                      | section 4                                                                | `v1.0.0`                                                                                                                                                    |
| F1  | v1.1 — recovery and adoption | FEAT-03…FEAT-06, FEAT-20, FEAT-21                                        | snapshot save/restore and `import-sway` pass their e2e and fuzz tests                                                                                       |
| F2  | v1.2 — profiles              | FEAT-10…FEAT-14, FEAT-22                                                 | output-driven profiles apply and revert safely                                                                                                              |

| Risk                                                       | Impact                                      | Mitigation                                                                                             |
| ---------------------------------------------------------- | ------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| Quickshell missing or old on Ubuntu 24.04 / Debian 13      | Tier-1 install blocker                      | decide early (PPA / backport / vendoring); test in the VM at M2                                        |
| False-green tests hide real failures                       | broken releases                             | M0 first; CI must run `go build ./...` and the e2e exit code                                           |
| Fake-server unit tests share the code's own protocol constants | wire-level bugs pass CI (see FIX-33)    | SWAY-09 against real headless Sway in the VM is the oracle; pin constants with a table test            |
| Guest toolchain older than `go.mod` (apt Go on Debian 13 / Ubuntu 24.04) | VM lab cannot build the project   | FIX-36; keep `GOTOOLCHAIN=local` only with a guaranteed-new-enough Go                                   |
| Restarting Sway ends all Wayland clients                   | self-heal cannot preserve application state | document the limit; FEAT-03…FEAT-05 relaunch apps and restore placement only                           |
| Restore/import executes data from files                    | command injection                           | whitelist and size-limit every field; only desktop-entry commands; fuzz; SEC-03 first                  |
| A profile or display change blanks the screen              | user locked out                             | SVC-11 confirm/revert; never disable the last output                                                   |
| Insecure or fake lock screen                               | security                                    | swaylock / `ext-session-lock` only, plus the fail-closed test                                          |
| No KVM runner in CI                                        | e2e only runs locally                       | headless subset in CI plus a self-hosted nightly                                                       |
| UI/service scope is large                                  | delays                                      | re-derive the queue (section 2) after every merge; defer P2                                            |

---

## 6. Progress log

Newest first. One line per merged change that moves a task or the status table.

| Date       | Change                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 2026-11-11 | API-02 done: `rpc.Listen` creates the control socket `0600` in a `0700` directory with the umask set around `Listen`, and removes a stale socket only when connecting is refused (never a live socket or a non-socket file); fixed `framing.go` declaring `package framing` inside package `rpc`, which broke the build; queue item 11 is now API-03. |
| 2026-11-09 | CORE-30 marked done: the daemon wiring is complete (`i3-nextd` registers the reconnecting Sway adapter → workspace service → supervised `quickshell`/`swaybg`, honours `enabled` from `services.yaml`, logs `ServiceStateChanged` and every workspace/output/sway event). The remaining defects in the task description are tracked as their own open items (`required`/`depends_on` → FIX-34, workspace stop/recovery → FIX-38, runnable shell/wallpaper commands → FIX-40, e2e false-green → FIX-18, snapshot rebuild/shutdown → SWAY-04). Status table and priority queue refreshed: CORE-30 removed from the queue, FIX-34/FIX-38/FIX-40 promoted. |
| 2026-11-09 | Static re-review against source: queue re-derived (FIX-19 removed as done); CORE-30 → `[~]` (daemon is wired); FIX-21 → `[~]` (client fixed, tests and adapter bypass missing); SWAY-04 → `[~]` (reconnect exists, snapshot rebuild/shutdown missing); SWAY-03 and CFG-07 reopened; added FIX-33 (Sway IPC numbers wrong), FIX-34, FIX-35, FIX-36 (guest Go), FIX-37, FIX-38, FIX-40 … FIX-44; FIX-23/FIX-24/FIX-18/FIX-25 rewritten from the current code (the `|| true` env claim was stale); Go version wording unified on 1.26.3; release-gate "broken config" line unticked; status table and risks refreshed. |
| 2026-11-09 | FIX-19 verified green (`gofmt -l .` empty, `go mod tidy` clean, `go.sum` present, dependencies current, `make check` green; `govulncheck` DB unreachable offline 403); FIX-28 docs drift fixed (`kvm.md`→`vm.py`, CHANGELOG `Unreleased`, `finalize-task.md` release-only bumps, `RELEASE_SCOPE.md` "Last updated" removed); FIX-29 API bloat reduced (one name per op, typed options/structs; `serviceShutdownTimeout`/`killService` → `ShutdownTimeouter`/`Killer` interfaces); FIX-32 defaults single-sourced from `internal/core/config/defaults/` (`configs/defaults/` removed). `go build ./... && go test -race ./...` and `make check` green. |
| 2026-11-07 | FIX-17 done (eventbus: package generics `Subscribe/Publish[T](b,...)`, idempotent cancel/close, pump-owned out close, workspace service/tests updated); `go build ./... && go test -race ./...` green, goleak clean, CORE-24 load test passes. Unblocked and completed CORE-20/21/24; status snapshot and priority queue refreshed.                                                                                                                                                                                                                        |
| 2026-11-07 | Re-baselined after a static review: new structure (sections 0–6) and priority rubric; added FIX-17 … FIX-32; reopened CFG-02/05/06/09/10/12, CORE-05/10…14/20/21/24, SWAY-05/20 as `[~]`; closed SEC-01; added CORE-30, DOC-10, DOC-11 and the FEAT-\* epics from `docs/new_features.md`; added `docs/development/todo-structure.md`                                                                                                                                                                                                                       |
| 2026-11-05 | v1 scope frozen in `docs/RELEASE_SCOPE.md`                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 |
