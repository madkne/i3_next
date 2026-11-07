---
name: golang-observability
description: "Instrument Go daemons and services with structured logs, metrics, tracing and safe diagnostics. Use for production logging, health/status reporting, reconnects, process supervision and performance signals."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Go Production Observability

**Persona:** You make production behavior diagnosable while avoiding secret leakage, unbounded cardinality and noisy logs.

## Signals
- Structured logs should include stable fields such as component, operation, outcome, duration and correlation/request ID.
- Log at the boundary where an error is handled; do not log and then return the same error at every layer.
- Distinguish expected lifecycle events (start, stop, reconnect) from warnings and failures.
- Expose health/readiness separately from detailed diagnostics. A process being alive does not mean Sway IPC or required services are ready.
- Track bounded metrics: event queue depth, dropped/coalesced events, reconnect attempts, process restart count, operation latency and service state.
- Never use unbounded labels such as raw command arguments, arbitrary paths, workspace names or user-controlled strings.
- Add tracing only where it helps follow work across asynchronous boundaries; propagate trace context.
- Protect diagnostics and profiling endpoints; do not expose pprof or sensitive state on an unauthenticated public listener.

## i3_next-specific guidance
- Include adapter/service names and lifecycle transitions in logs.
- Log process exit status and restart reason without dumping environment variables or full command secrets.
- Make reconnect loops observable with attempt count and backoff duration.
- Ensure logs are useful even when the compositor is unavailable.

## Cross-references
See `golang-context`, `golang-security`, `golang-troubleshooting` and `golang-benchmark`.
