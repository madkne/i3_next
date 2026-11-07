# Go skills for i3_next

These skills complement the existing Go skills in this directory. They are adapted
from the `samber/cc-skills-golang` collection and tailored to this repository.

## Added skills

| Skill                           | Use for                                                       |
| ------------------------------- | ------------------------------------------------------------- |
| `golang-benchmark`              | Benchmarking, pprof, traces and performance comparisons       |
| `golang-cli`                    | `i3-nextd`, `i3-nextctl`, `i3-next-session`, `i3-next-doctor` |
| `golang-concurrency`            | Goroutines, event bus, reconnects, workers and shutdown       |
| `golang-continuous-integration` | CI quality gates and release workflows                        |
| `golang-dependency-injection`   | Constructor wiring and testable boundaries                    |
| `golang-dependency-management`  | Go modules, versions, licenses and supply-chain review        |
| `golang-lint`                   | golangci-lint, vet and static analysis                        |
| `golang-modernize`              | Safe modernization for the Go 1.24 baseline                   |
| `golang-naming`                 | Idiomatic identifiers and repository vocabulary               |
| `golang-observability`          | Structured logs, health, metrics and diagnostics              |
| `golang-performance`            | Profile-guided optimization                                   |
| `golang-popular-libraries`      | Dependency selection and standard-library-first decisions     |
| `golang-project-layout`         | Package boundaries and import direction                       |
| `golang-refactoring`            | Behavior-preserving changes                                   |
| `golang-safety`                 | Runtime, memory, filesystem and resource safety               |
| `golang-structs-interfaces`     | Types, interfaces and adapter contracts                       |
| `golang-testing`                | Unit, integration, fuzz and race tests                        |

## Existing skills retained

`golang-code-style`, `golang-context`, `golang-data-structures`,
`golang-design-patterns`, `golang-documentation`, `golang-error-handling`,
`golang-security` and `golang-troubleshooting` remain unchanged.

## Project constraints

- `go.mod` currently targets Go 1.26.3. Do not use APIs requiring a newer Go version
  without an explicit toolchain decision.
- Follow `docs/architecture/OVERVIEW.md` and `docs/development/dependency-policy.md`.
- Do not run Sway on the host. All compositor-dependent verification belongs in the
  KVM guest as specified by `.kilo/rules/no-run-sway.md`.
- Prefer focused changes and the existing `make check` workflow.
