---
name: golang-dependency-management
description: "Manage Go modules, versions, upgrades, tool dependencies and supply-chain risk. Use when adding, upgrading, replacing or removing Go modules or build tools."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Go Dependency Management

**Persona:** You treat every dependency as a maintenance, licensing and supply-chain commitment.

## Project baseline

The module is `i3_next` and `go.mod` currently declares Go 1.26.0. Read the current module file and dependency policy before making changes.

## Rules

- Prefer the standard library when it meets the requirement.
- Before adding a module, explain the capability gap, maintenance status, license, transitive footprint and security implications.
- Avoid adding a package for a trivial helper or a feature already covered by an existing dependency.
- Use `go get module@version` intentionally; inspect the resulting `go.mod` and `go.sum` diff.
- Run `go mod tidy` and verify it produces no unexpected changes.
- Do not use floating versions in reproducible build scripts; pin tool versions where practical.
- Avoid `replace` directives except for a documented temporary fork/local development need.
- Review transitive dependency changes and check license compatibility with the repository's MIT license.
- Do not upgrade the `go` directive or toolchain requirement merely because a newer version exists; check supported distributions and CI first.
- Never vendor or commit generated dependency caches.

## Validation

```sh
go mod tidy
git diff -- go.mod go.sum
go list -m all
go test ./...
```

Run vulnerability scanning when available and record any accepted exception with a reason and expiry/review condition.

## Cross-references

See `golang-popular-libraries`, `golang-security`, `golang-continuous-integration` and `golang-modernize`.
