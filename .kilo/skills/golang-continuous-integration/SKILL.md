---
name: golang-continuous-integration
description: "Design and maintain Go CI quality gates, security scanning, tests, linting and release workflows with GitHub Actions. Use when changing workflows, Makefile checks or release automation."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Go Continuous Integration

**Persona:** CI is a reproducible quality gate, not a place to hide flaky tests or silently skip checks.

## Required project gates
- `gofmt` / formatting check
- `go vet ./...`
- `golangci-lint run ./...` when installed/configured
- `staticcheck ./...` when part of the project's supported toolchain
- `go test -race ./...` for host-safe unit tests
- `scripts/check-naming.sh`
- Build all Go binaries for the supported target
- Dependency and vulnerability review (`govulncheck` when available)

Use the repository's existing `make check` and CI workflow as the source of truth. Do not invent parallel scripts that duplicate them.

## Workflow design
- Pin GitHub Actions to stable major versions or immutable SHAs according to repository policy.
- Set least-privilege `permissions` per job.
- Use Go version from `go.mod` (currently Go 1.24.0); avoid testing unsupported toolchains accidentally.
- Cache Go modules/build artifacts safely; do not cache generated secrets or mutable release outputs.
- Run independent checks in parallel, but preserve actionable logs and avoid hiding failures behind `continue-on-error`.
- Use `-count=1` for integration tests when cached results would invalidate the check.
- Keep VM/Sway checks separate from host-safe jobs and execute them only in the supported KVM guest workflow.
- Release jobs must be gated on tests, lint, build and security checks; use protected environments for publishing.

## Security settings
After adding workflows, tell the maintainer to configure branch protection, required status checks, workflow token permissions, secrets and protected environments in GitHub repository settings. Do not assume YAML alone enables repository protection.

## Cross-references
See `golang-testing`, `golang-lint`, `golang-security`, `golang-dependency-management` and `golang-cli`.
