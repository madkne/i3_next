---
name: golang-project-layout
description: "Design and review Go repository structure, package boundaries, cmd/internal layout, dependency direction and build organization. Use when adding packages, moving code or changing architecture."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Go Project Layout

**Persona:** You organize code around clear responsibilities and dependency direction, not fashionable folder templates.

## Repository conventions
- Keep executable entry points under `cmd/<binary>/`.
- Put implementation packages under `internal/` unless external reuse is an explicit requirement.
- Use `core` for framework/OS-independent logic; place Sway IPC and OS integrations in adapters.
- Keep package boundaries meaningful. Do not create empty packages or folders in anticipation of future code.
- Avoid catch-all `pkg/`, `utils/` and `common/` packages.
- Keep `main.go` small and perform dependency construction at the composition root.
- Avoid import cycles; move shared contracts to the narrowest appropriate layer.
- Prefer package names that describe a cohesive capability and keep exported APIs small.

## i3_next architecture
Follow `docs/architecture/OVERVIEW.md` and the documented layering: `core <- adapters/services <- api`. Do not make core depend on Sway, Quickshell, Qt, process execution or host filesystem implementations. Keep UI/QML integration at the boundary.

Before moving code, inspect imports, tests, build tags and external package references. Update architecture documentation when a user-visible boundary or package responsibility changes.

## Cross-references
See `golang-dependency-injection`, `golang-structs-interfaces`, `golang-documentation` and `golang-refactoring`.
