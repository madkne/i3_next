---
name: golang-dependency-injection
description: "Design testable Go components using constructor injection and small consumer-owned interfaces. Use when wiring daemon services, adapters, process runners, event buses or refactoring global state."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Dependency Injection in Go

**Persona:** You prefer explicit construction and small interfaces over service locators, global registries or framework-driven wiring.

## Rules
1. Inject dependencies through constructors; avoid mutable package globals and service setup in `init`.
2. Define interfaces at the consuming package, with only the methods that consumer needs.
3. Return concrete types from constructors unless an interface return is an intentional API boundary.
4. Keep the dependency graph visible at the application composition root (`cmd/i3-nextd` / bootstrap).
5. Do not pass a general-purpose container through application code. Inject the specific service required.
6. Use manual constructor injection by default. The current project does not justify a DI framework merely to wire a modest service graph.
7. Make lifecycle ownership explicit: who starts, stops and closes each dependency?
8. For OS/process/IPC behavior, inject narrow interfaces so unit tests can use fakes without launching Sway or real child processes.

## i3_next guidance
- Follow the existing layering and dependency direction documented in `docs/architecture/OVERVIEW.md`.
- Keep core packages independent of Sway, Quickshell, filesystem and process implementations.
- Keep adapter construction in bootstrap/composition code.
- Do not create abstractions for one-off values or methods that have no second consumer.
- Avoid passing `*App` or a large service container to every package; pass the exact capability required.

## Review checklist
- Are dependencies explicit and constructor-validated?
- Are interfaces small and owned by consumers?
- Can unit tests replace external effects?
- Is lifecycle ownership unambiguous?
- Are import cycles avoided?

## Cross-references
See `golang-structs-interfaces`, `golang-design-patterns`, `golang-testing` and `golang-project-layout`.
