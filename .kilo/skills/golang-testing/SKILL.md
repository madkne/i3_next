---
name: golang-testing
description: "Write reliable Go unit, integration, fuzz and concurrency tests. Use for behavior changes, regressions, test architecture, race detection and flaky-test diagnosis."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Go Testing Best Practices

**Persona:** Tests are executable specifications of observable behavior, not a coverage contest.

## Rules
- Use table-driven tests with named subtests for multiple cases.
- Assert public behavior and contracts rather than private implementation details.
- Keep unit tests deterministic, fast and independent of execution order.
- Use `t.Parallel()` only when fixtures and shared state are truly isolated.
- Mock interfaces at boundaries; do not mock every internal function.
- Use `t.TempDir()` and cleanup helpers for filesystem tests.
- Test error paths, cancellation, partial writes, malformed input and resource cleanup.
- Add fuzz tests for parsers, IPC framing, config decoding and other untrusted-input boundaries.
- Run `go test -race ./...`; use goleak for packages that own long-lived goroutines.
- Put environment-dependent integration tests behind `//go:build integration` and document their prerequisites.
- Do not make host unit tests launch Sway or depend on a live Wayland session.

## i3_next test strategy
- Unit-test core logic with fakes for IPC, process execution, clocks and filesystem.
- Test Sway IPC encoding/decoding with captured fixtures and fuzzing.
- Test reconnect, shutdown and supervisor behavior with deterministic fake transports/processes.
- Run compositor-dependent tests, screenshots, hotplug and end-to-end flows only inside the KVM guest, following `.kilo/rules/no-run-sway.md`.
- Ensure tests do not leave child processes, sockets, goroutines or temporary files behind.

## Useful commands
```sh
go test ./...
go test -race ./...
go test -shuffle=on -count=1 ./...
go test -run '^$' -bench . -benchmem ./...
go test -fuzz FuzzName -fuzztime 30s ./path
```

## Cross-references
See `golang-concurrency`, `golang-benchmark`, `golang-cli`, `golang-continuous-integration` and `golang-troubleshooting`.
