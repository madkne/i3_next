---
name: golang-naming
description: "Apply idiomatic Go naming to packages, types, interfaces, methods, errors, constants, receivers and tests. Use when creating or reviewing identifiers and public APIs."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Go Naming Conventions

**Persona:** You choose names that make code readable at call sites and consistent across the repository.

## Rules
- Use MixedCaps; never underscores in Go identifiers. Package names are short, lowercase and normally one word.
- Avoid generic packages named `util`, `common`, `helpers` or `misc`; name packages after their responsibility.
- Use `New` / `NewType` for constructors, `WithX` for functional options, `ErrX` for sentinel errors and `XError` for error types.
- Use standard initialisms consistently: `ID`, `URL`, `HTTP`, `IPC`, `OS`, `UI`.
- Keep interface names behavior-oriented (`Reader`, `Runner`, `Dialer`) and small.
- Avoid redundant Get prefixes for simple getters; use `Is`, `Has`, `Can` for predicates.
- Receiver names should be short and consistent across methods on the same type.
- Name test cases by behavior, not implementation detail; use descriptive subtest names.
- Preserve established public names unless a change provides clear value; exported renames may be breaking changes.

## i3_next vocabulary
Use consistent terms for daemon, session, workspace, output, event, adapter, supervisor, process and service. Do not mix `session` and `workspace` when they represent distinct concepts. Follow `docs/development/naming.md` and `scripts/check-naming.sh`.

## Cross-references
See `golang-code-style`, `golang-structs-interfaces` and `golang-lint`.
