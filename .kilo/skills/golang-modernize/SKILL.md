---
name: golang-modernize
description: "Modernize Go code safely using language and standard-library improvements supported by the module's Go version. Use for Go upgrades, deprecations, idiom improvements and modernization reviews."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Go Modernization

**Persona:** You modernize incrementally. Correctness and security come before readability, and readability comes before novelty.

## Compatibility rule

The repository currently targets Go 1.26.0. Use only APIs available at that minimum unless the project deliberately raises its minimum version after checking CI, packaging and target distributions. Never copy examples requiring Go 1.25/1.26 without checking availability.

## Candidates

- Prefer `any`, `slices`, `maps`, `cmp`, typed atomics and `log/slog` where they simplify code.
- Use `range` over integers and `t.Context()` / `b.Loop()` only when supported by the declared Go version.
- Replace deprecated APIs only when behavior and compatibility are understood.
- Evaluate `os.Root` for operations involving untrusted filesystem paths.
- Keep iterators only when they improve clarity and avoid unnecessary materialization.
- Do not adopt experimental APIs (for example experimental JSON packages) in production paths without an explicit decision.

## Safe workflow

1. Identify a focused modernization opportunity.
2. Add or confirm behavior tests before changing implementation.
3. Change one concern at a time; do not mix broad refactoring with a feature task.
4. Run tests, race checks and static analysis.
5. Explain compatibility impact and avoid raising `go.mod` without approval.

## Cross-references

See `golang-testing`, `golang-refactoring`, `golang-performance`, `golang-safety` and `golang-dependency-management`.
