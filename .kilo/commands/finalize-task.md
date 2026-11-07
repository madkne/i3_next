---
description: Finalize current task, update TODO, CHANGELOG and version
---

# Task Finalization Workflow

Please execute the following steps in order:

1. **Update TODO.md**: Read the current TODO.md, mark the completed task items as done (or delete them), and add any newly discovered pending items.
2. **Update CHANGELOG.md**: In the "Unreleased" or latest version section, add the changes for this task, following the existing format of the file.
3. **Update version**: Read and increment the version number in `internal/version/version.go` (default to patch version +1). If the file contains a version string constant, modify it directly.

After all three steps are completed, summarize which files were modified.