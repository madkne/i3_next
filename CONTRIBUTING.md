# Contributing to i3_next

Thank you for your interest in contributing. This document describes how to
set up a development environment, the standards we expect, and the process for
submitting changes.

## Prerequisites

- Linux (Ubuntu LTS 24.04 or Debian Stable 13 are the primary targets)
- Go 1.24.0 or newer
- Git
- `make`
- Optional but recommended: `golangci-lint`, `staticcheck`, `shellcheck`

Install development dependencies:

```bash
./scripts/dev-setup.sh
```

This downloads Go modules, configures the repository's git hooks, installs the
lint/static-analysis tools, and builds the project.

## Development workflow

1. Create a feature branch from the default branch.
2. Make your changes in small, focused commits.
3. Run the full local quality gate before opening a pull request:

   ```bash
   make check
   ```

   `make check` runs, in order:

   - `fmt-check` — `gofmt` formatting check
   - `vet` — `go vet ./...`
   - `lint` — `golangci-lint run ./...`
   - `staticcheck` — `staticcheck ./...`
   - `test` — `go test -race ./...`

4. Additional checks:

   ```bash
   make shellcheck   # shell scripts
   make qmllint      # QML files (when qmllint is available)
   ```

5. Open a pull request describing the change, its motivation, and how it was
   tested.

## Code standards

- All Go code must be formatted with `gofmt` / `goimports`. CI rejects
  unformatted code.
- New code must pass `go vet`, `golangci-lint`, and `staticcheck` without
  new findings.
- Shell scripts must pass `shellcheck`.
- QML must pass `qmllint` when the tool is available.
- Keep packages focused and avoid importing UI packages from core packages.
- Prefer small, reviewable pull requests.

## Tests

- Add or update tests for every behavior change.
- Run `go test -race ./...` locally; CI runs the same command.
- Do not commit code that breaks existing tests.

## Commit messages

Use concise, imperative commit messages, e.g. `add CONTRIBUTING and SECURITY
docs`. Reference relevant issues when applicable.

## Reporting issues

Use the project issue tracker. Include the output of `i3-next-doctor` and the
relevant logs when reporting bugs.

## License

By contributing, you agree that your contributions are licensed under the
project's MIT License (see `LICENSE`).
