---
name: golang-safety
description: "Prevent Go runtime defects and silent corruption: nil handling, slice aliasing, map concurrency, integer overflow, resource leaks, filesystem boundaries and unsafe usage."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Defensive Go Programming

**Persona:** You assume inputs, lifetimes and ownership can be wrong until validated at the boundary.

## Rules
- Validate external data, IPC payloads, config files and process output before using them.
- Design useful zero values; initialize maps before writes and avoid accidental nil slices in JSON contracts.
- Understand slice backing-array aliasing. Clone data when ownership must be independent.
- Never access a map concurrently without synchronization.
- Check integer arithmetic and size conversions when parsing lengths, dimensions, timeouts or protocol fields.
- Avoid `unsafe`; if unavoidable, document the invariant and keep the unsafe region minimal.
- Close files, sockets, pipes and response bodies. Handle cleanup errors where they affect correctness.
- Do not use `defer` in an unbounded loop when it delays resource release; close each iteration's resource before continuing.
- Treat filesystem paths from config, IPC or user input as untrusted. Prevent traversal and symlink escapes where the operation crosses a trust boundary.
- Bound allocations based on validated input sizes; reject oversized IPC/config payloads.
- Avoid panic for expected failures; return contextual errors.

## i3_next-specific checks
- Validate Sway IPC frame lengths and message types before allocation/decoding.
- Avoid trusting process names, PIDs, environment values or filesystem notifications as authoritative without revalidation.
- Ensure config writes are atomic and preserve permissions.
- Do not leak environment variables, tokens or private paths in logs.

## Cross-references
See `golang-security`, `golang-error-handling`, `golang-data-structures` and `golang-concurrency`.
