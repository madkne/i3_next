---
name: golang-lint
description: "Configure and interpret Go linters, especially golangci-lint, staticcheck, go vet and security-focused checks. Use when changing .golangci.yml, fixing diagnostics or adding quality gates."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Go Linting

**Persona:** You use linters to expose defects and enforce agreed conventions, not to maximize the number of enabled rules.

## Project rules
- Inspect the existing `.golangci.yml` before changing configuration.
- Keep lint configuration compatible with the version used in CI and developer setup.
- Prefer fixing the underlying issue over suppressing a diagnostic.
- Every `//nolint` must name the linter and explain why the finding is a false positive or intentional trade-off.
- Do not mass-enable a large preset and then suppress hundreds of findings.
- Avoid conflicting formatters/linters and keep formatting deterministic.
- Review security/resource checks such as unchecked errors, HTTP body closure, goroutine leaks, unsafe file paths and subprocess handling.
- Keep generated code exclusions narrow and documented.

## Workflow
1. Reproduce the diagnostic with the same tool version as CI.
2. Read the linter's rule and understand the actual risk.
3. Fix the code or add a narrowly scoped, justified suppression.
4. Run the affected package, then the full configured lint command.
5. Check that the config itself validates.

## Cross-references
See `golang-code-style`, `golang-security`, `golang-continuous-integration` and `golang-testing`.
