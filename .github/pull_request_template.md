<!--
  i3_next pull request template.
  Before opening, read CONTRIBUTING.md and run `make check`.
-->

## Summary
Briefly describe the change and its motivation.

## Linked issue
Closes # (or `none` if there is no issue).

## Dependency changes
- New Go dependencies added: none / list module@version
  - Reason each is needed:
  - License of each (compatible with MIT): see `docs/development/dependency-policy.md`
- No dependency changes

## Testing
- `go test -race ./...` passes
- `make check` passes (fmt, vet, golangci-lint, staticcheck, test, naming)
- Unit/integration tests added or updated for the changed behavior
- e2e/VM test outcome (if applicable)

## Checklist
- [ ] Commits use conventional, imperative messages
- [ ] No new empty package folders (STRUCT-01: packages only when they have real code)
- [ ] Import layering respected: `core <- adapters/services <- api` (enforced by depguard in `.golangci.yml`)
- [ ] Naming rule followed (`i3_next` module/Go, `i3-next` dirs/binaries; see `docs/development/naming.md`)
- [ ] CHANGELOG.md updated with an `Unreleased` entry
- [ ] Docs updated (`README`, `docs/`), if user-facing

## Screenshots / logs
(optional) Relevant output or `i3-next-doctor` output for UI/session changes.
