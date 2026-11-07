---
name: golang-structs-interfaces
description: "Design Go structs and interfaces: composition, embedding, serialization tags, method sets, pointer/value receivers, type assertions and consumer-owned interfaces."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Go Structs & Interfaces

**Persona:** You use types to make contracts explicit and keep abstractions small.

## Interface design
- Define interfaces where they are consumed, not beside the implementation by default.
- Prefer 1-3 method interfaces that describe the capability required by a caller.
- Accept interfaces and return concrete structs in most constructors.
- Do not introduce an interface for a type with one implementation unless it creates a real test or architectural boundary.
- Avoid `any` when a type parameter or concrete type expresses the contract.
- Add compile-time checks (`var _ Interface = (*Concrete)(nil)`) for important adapter implementations.
- Handle type assertions with the comma-ok form and return meaningful errors for unsupported values.

## Struct design
- Use named fields in composite literals.
- Keep serialized field tags explicit and stable; test JSON/YAML contracts.
- Choose pointer receivers when mutation, identity or large struct size warrants them; be consistent across a type's method set.
- Avoid embedding solely to save a few forwarding methods; composition with a named field may be clearer.
- Make invalid states difficult to represent and document ownership of mutable slices/maps.

## i3_next boundaries
Use small interfaces around Sway IPC connections, process execution, clocks, filesystem access and event subscribers when tests need to replace those effects. Do not expose adapter-specific types in core contracts.

## Cross-references
See `golang-dependency-injection`, `golang-naming`, `golang-code-style` and `golang-testing`.
