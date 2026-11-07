---
name: golang-refactoring
description: "Refactor existing Go code safely with behavior-preserving, incremental changes. Use for package moves, API cleanup, decomposition, deduplication and architectural improvements."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Safe Go Refactoring

**Persona:** You protect existing behavior first. A refactor changes structure, not externally observable behavior.

## Workflow
1. Read the relevant implementation, callers, tests and architecture docs before editing.
2. Define the behavior and compatibility surface that must remain unchanged.
3. Establish a test/build baseline; add characterization tests when behavior is undocumented.
4. Make small, reviewable steps. Keep unrelated formatting or feature changes out of the refactor.
5. Use compiler-aware tools (`gopls` rename/extract/inline) for symbol changes; inspect every resulting diff.
6. Avoid broad textual replacements for identifiers or package moves.
7. Run focused tests after each step, then `go test -race ./...`, vet and lint.
8. Update docs only when the architecture or user-facing contract actually changes.

## i3_next safety constraints
- Preserve CLI flags, exit codes, config compatibility, event contracts and Sway IPC behavior.
- Do not run Sway on the host. Compositor-dependent verification belongs in the KVM guest.
- Keep the no-host-Sway rule in `.kilo/rules/no-run-sway.md` authoritative.
- Do not combine refactoring with unrelated new functionality unless explicitly requested.

## Review checklist
Check import direction, lifecycle ownership, goroutine shutdown, error wrapping, config migrations, public API compatibility and test coverage.

## Cross-references
See `golang-testing`, `golang-project-layout`, `golang-modernize` and `golang-troubleshooting`.
