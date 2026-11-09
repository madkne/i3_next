---
name: golang-popular-libraries
description: "Evaluate Go libraries and frameworks before adding dependencies. Use when selecting packages for CLI, IPC, config, logging, testing, process management or platform integration."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Go Libraries & Framework Selection

**Persona:** You know the Go ecosystem but prefer the standard library when it is sufficient.

## Decision criteria
1. Does the standard library already solve the problem?
2. Is the dependency actively maintained and compatible with the Go version in `go.mod`?
3. Is its license compatible with this MIT-licensed project?
4. What are its transitive dependencies, binary-size impact and attack surface?
5. Does it fit Linux/Wayland packaging and the supported distributions?
6. Can it be isolated behind a small interface and tested without Sway?
7. Does it introduce global state, reflection, code generation or lifecycle complexity?

## i3_next guardrails
- Do not add a second config format/parser when YAML support already exists.
- Do not add a process-management framework for simple `os/exec` use cases.
- Do not add a DI container for the current service graph without demonstrated lifecycle complexity.
- Prefer narrow, maintained IPC and filesystem dependencies only where standard-library code would be less correct or materially harder to maintain.
- Avoid framework-specific skills/dependencies for Cobra, Viper, gRPC, GraphQL or database stacks unless the project actually adopts them.
- Record the reason and license for each new direct dependency in the relevant PR/change documentation.

## Cross-references
See `golang-dependency-management`, `golang-security`, `golang-cli` and `golang-testing`.
