---
name: golang-performance
description: "Apply measured Go performance improvements to CPU, memory, allocations, GC, I/O and concurrency hot paths. Use after profiling or when reviewing a known bottleneck."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Go Performance Optimization

**Persona:** You optimize only measured bottlenecks and preserve correctness, responsiveness and resource bounds.

## Workflow
1. Define the metric and capture a reproducible baseline.
2. Profile to identify the actual hot path.
3. Form one hypothesis and make one focused change.
4. Re-run the same benchmark/profile and compare results.
5. Keep the change only when the improvement is repeatable and does not regress other important metrics.

## Areas to inspect
- Avoid unnecessary allocations and repeated serialization in event/IPC paths.
- Reuse buffers only when ownership and concurrency safety are explicit; reset pooled objects before reuse.
- Bound queues and process counts; unbounded buffering is a reliability defect, not a performance optimization.
- Reduce lock contention by shortening critical sections and avoiding I/O while locked.
- Prefer efficient data structures based on access patterns, not microbenchmark folklore.
- Stream large inputs instead of loading unbounded data into memory.
- Avoid premature caching; define invalidation and memory limits before adding a cache.
- Consider GC tuning or PGO only after production profiles demonstrate a need.

## i3_next constraint
Do not optimize compositor integration by running Sway on the host. Use unit benchmarks for isolated logic and the KVM guest for compositor-dependent measurements.

## Cross-references
See `golang-benchmark`, `golang-data-structures`, `golang-concurrency`, `golang-observability` and `golang-troubleshooting`.
