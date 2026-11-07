---
name: golang-concurrency
description: "Design and review Go goroutines, channels, locks, worker pools, event pipelines and cancellation. Use for daemon lifecycle, event bus, filesystem watcher, Sway IPC reconnect and process supervision code."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Go Concurrency Best Practices

**Persona:** You assume every goroutine is a resource with an owner, a shutdown path and an observable failure mode.

## Core rules
1. Every goroutine must have a clear owner, cancellation mechanism and join/wait path.
2. Propagate the same `context.Context`; include `ctx.Done()` in blocking selects.
3. Prefer synchronous code until concurrency solves a measured problem.
4. Use channels to transfer ownership; use mutexes for shared state. Do not mix ownership models casually.
5. Only the sender/owner closes a channel. Document channel direction in function signatures.
6. Bound worker counts, queues, retries and pending work. Define overload/backpressure behavior.
7. Never hold a mutex while performing IPC, filesystem, process or other blocking I/O.
8. Do not spawn one goroutine per event/process without a limit.
9. Use `errgroup` when a group needs error propagation, cancellation or concurrency limits; use `WaitGroup` for simple joins.
10. Reuse timers in long-running loops rather than repeatedly allocating `time.After`.

## i3_next-specific checks
- Daemon shutdown must stop watchers, event consumers, reconnect loops, child processes and supervisors before dependencies are closed.
- Sway reconnect must not create duplicate reader goroutines or duplicate subscriptions.
- Event bus subscribers need bounded queues and an explicit policy for slow consumers (drop, coalesce, block or disconnect); document the choice.
- Process supervision must distinguish intentional restart from shutdown and prevent restart storms with bounded backoff.
- Config reload and filesystem notifications must be coalesced/debounced without losing the final state.
- Every goroutine started by a service must be accounted for in its `Start`/`Stop` lifecycle.

## Before spawning a goroutine
- Who owns it?
- What ends it?
- Who waits for it?
- How are errors surfaced?
- What bounds its resource use?
- What happens under cancellation and backpressure?

## Verification
Run `go test -race ./...` and add deterministic shutdown/reconnect tests. Use goleak where appropriate. Sway-dependent tests must run inside the KVM guest, never on the host.

## Cross-references
See `golang-context`, `golang-safety`, `golang-testing`, `golang-observability` and `golang-troubleshooting`.
