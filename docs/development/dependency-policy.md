# Dependency policy

i3_next keeps its dependency surface small and auditable. Every dependency in
`go.mod` must be justifiable, licensed compatibly with the project's MIT
license, and recorded in `go.sum`.

## Rules

1. **Every new dependency needs a reason in the PR.**
   A pull request that adds, upgrades, or removes a Go module must state, for
   each affected module:
   - the module and version,
   - why it is needed (what feature it enables),
   - why the existing dependencies cannot satisfy the need,
   - whether it is a production or test-only dependency.

2. **License check.**
   Every new dependency must be reviewed for license compatibility with the
   project's MIT license (`LICENSE`). Copyleft (GPL/AGPL) and non-commercial
   licenses require explicit review and approval before being merged. The result
   of each review is recorded in `THIRD_PARTY_LICENSES.md` (see below).

3. **`go.sum` stays in sync.**
   `go.sum` must match `go.mod`. Run `go mod verify` locally and keep `go.sum`
   committed with every dependency change. CI builds with
   `GOFLAGS=-mod=readonly` (see `.golangci.yml`).

4. **Prefer the standard library.**
   Do not add a dependency for functionality the Go standard library already
   provides.

5. **Vulnerability scanning.**
   `govulncheck ./...` runs in CI. A new dependency must not introduce an
   unresolved `critical` or `high` vulnerability (see `SEC-04`, `FIX-12`).

## Records

- `THIRD_PARTY_LICENSES.md` — generated list of all dependencies and their
  licenses (see `SEC-10`). Regenerate it whenever `go.mod` changes.
- `SECURITY.md` — supply-chain and vulnerability disclosure policy
  (`SEC-09`, `SEC-12`).

## Related tasks

- `STRUCT-03` — `go.sum` and this dependency policy (⛓ `SEC-10`)
- `SEC-10` — `THIRD_PARTY_LICENSES.md` generated with `go-licenses` and a license compatibility check
- `FIX-12` / `SEC-04` — pinned tools, `govulncheck` in CI
