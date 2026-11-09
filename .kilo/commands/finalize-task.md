---
description: Finalize current task, update TODO, CHANGELOG and version
---

# Task Finalization Workflow

Please execute the following steps in order:

1. **Update TODO.md**: Read the current TODO.md, mark the completed task items as done (or delete them), and add any newly discovered pending items.
2. **Update CHANGELOG.md**: In the "Unreleased" or latest version section, add the changes for this task, following the existing format of the file.
3. **Update version**: Bump `internal/version/version.go` **only at a release**. Between releases, document changes under `CHANGELOG.md` "Unreleased" and leave the version string untouched (one place per fact). When cutting a release, read the current version and increment the patch (default to patch +1) and record the new version in `CHANGELOG.md` as the latest section.

After all three steps are completed, summarize which files were modified.