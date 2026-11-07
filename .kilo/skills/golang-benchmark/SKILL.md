---
name: golang-benchmark
description: "Measure Go performance with benchmarks, pprof, execution traces and statistically sound comparisons. Use before proposing performance optimizations or when investigating CPU, memory, allocation or latency regressions."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Go Benchmarking & Performance Measurement

**Persona:** You are a performance measurement engineer. Never claim an optimization worked from intuition or one noisy run.

## Workflow
1. Identify the user-visible performance symptom and define a measurable metric.
2. Locate the relevant code path and add a representative benchmark before changing it.
3. Separate fixture/setup work from the timed section; use `b.Loop()` (Go 1.24+) where suitable.
4. Call `b.ReportAllocs()` and record throughput/latency where meaningful.
5. Run repeated samples and compare with `benchstat`; keep machine, Go version, CPU and workload consistent.
6. Capture CPU, heap or execution-trace profiles when the benchmark does not explain the bottleneck.
7. Change one thing at a time, rerun the same workload, and report both improvements and regressions.

## Commands
```sh
go test -run '^$' -bench . -benchmem -count=10 ./...
go test -run '^$' -bench BenchmarkName -cpuprofile cpu.out ./path
go tool pprof cpu.out
go test -run '^$' -bench BenchmarkName -memprofile mem.out ./path
go tool trace trace.out
```

## Rules
- Benchmark realistic Sway IPC payloads, config sizes, event rates and process counts; synthetic microbenchmarks alone are not sufficient.
- Do not benchmark compositor-dependent behavior on the host. Use the KVM guest for Sway integration measurements.
- Do not add benchmark thresholds to shared CI until runner variance is understood.
- Never optimize without a baseline, and never trade correctness, cancellation or bounded resource use for a small benchmark win.
- Keep benchmark fixtures deterministic and avoid network, disk or VM setup inside the timed loop unless that is what is being measured.

## Cross-references
See `golang-performance` for optimization techniques, `golang-observability` for production signals, `golang-troubleshooting` for live diagnosis and `golang-testing` for test organization.
