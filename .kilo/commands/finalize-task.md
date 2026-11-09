---
description: Finalize current task, update TODO, CHANGELOG and version
---

# Task Finalization Workflow

Please execute the following steps in order:

1. **Update TODO.md**: Read the current TODO.md, mark the completed task items as done (or delete them), and add any newly discovered pending items.
2. **Update CHANGELOG.md**: In the "Unreleased" or latest version section, add the changes for this task, following the existing format of the file.
3. **Update version**: Bump `internal/version/version.go` **only at a release**. Between releases, document changes under `CHANGELOG.md` "Unreleased" and leave the version string untouched (one place per fact). When cutting a release, read the current version and apply the increment rule below, then record the new version in `CHANGELOG.md` as the latest section.

## Version Change Rules

This project follows [Semantic Versioning](https://semver.org/) (`MAJOR.MINOR.PATCH`). When cutting a release, decide the bump based on the changes accumulated under "Unreleased" in `CHANGELOG.md`:

| Bump | When to use | Examples |
|------|-------------|----------|
| **MAJOR** (`X.0.0`) | Incompatible or breaking changes to public API, CLI flags, config format, data/protocol format, or behavior users depend on. | Removing/renaming a public function; changing a CLI flag's meaning; dropping a supported file format; requiring a new mandatory config field. |
| **MINOR** (`x.Y.0`) | Backward-compatible new functionality. | Adding a new command, flag, config option, public API function, or output format that existing users can ignore. |
| **PATCH** (`x.y.Z`) | Backward-compatible bug fixes and non-functional changes. | Bug fixes, performance improvements, refactors, docs, dependency bumps without behavior change, internal tooling. |

**Decision rules:**

- If *any* change in the release is breaking → **MAJOR** (regardless of other changes).
- Else if *any* change adds user-facing functionality → **MINOR**.
- Else → **PATCH** (the default).
- While the project is at `0.x.y`, MAJOR bumps are still reserved for breaking changes; per SemVer, breaking changes in `0.x` bump MINOR, but this project opts to treat them as MAJOR to keep signals explicit — call this out in the CHANGELOG if you deviate.
- Pre-release identifiers (e.g. `-rc.1`) may be appended when staging a release; strip them for the final release.
- **One version, one place:** `internal/version/version.go` is the single source of truth; `CHANGELOG.md` mirrors it for humans. Never bump the version outside a release step.

After all three steps are completed, summarize which files were modified (and, if a release was cut, state the old → new version and the bump type applied).