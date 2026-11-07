---
name: golang-cli
description: "Build and review Unix-friendly Go command-line programs: flags, configuration, exit codes, signals, stdout/stderr, version metadata and CLI tests. Use for i3-nextctl, i3-next-doctor and session command changes."
license: MIT
source: https://github.com/samber/cc-skills-golang
---

# Go CLI Engineering

**Persona:** You build predictable, scriptable Unix tools that behave well in terminals, shell pipelines and automation.

## Project-specific context
The repository has separate binaries under `cmd/`: `i3-nextd`, `i3-nextctl`, `i3-next-session` and `i3-next-doctor`. Read the existing command implementation before proposing Cobra/Viper or another framework. Do not introduce a CLI framework for a small command with a few flags; the standard library is preferred unless complexity justifies a dependency.

## Rules
- Keep `main` thin: parse configuration, construct dependencies, invoke the application, map errors to process exit codes.
- Keep stdout for requested command output and stderr for diagnostics/logging. Support stable machine-readable output when a command is used by scripts.
- Validate arguments and configuration at the boundary; errors should identify the invalid flag/value without dumping secrets.
- Use `signal.NotifyContext` for SIGINT/SIGTERM and propagate cancellation to all work.
- Never call `os.Exit` deep inside command handlers; return errors so deferred cleanup runs.
- Use conventional exit codes: 0 success, non-zero failure; distinguish usage errors from runtime failures where useful.
- Embed build metadata through the existing Makefile/build scripts. Preserve the project's `build_date` and `commit` convention; do not replace it with a hard-coded version.
- Make commands testable by injecting readers, writers, process runners and service interfaces instead of writing directly to global stdio.
- Do not add Cobra/Viper, color, prompt or table dependencies without a concrete user-facing need and dependency-policy review.

## Review checklist
- Are defaults and precedence documented?
- Are errors and logs sent to stderr?
- Does cancellation stop child processes?
- Are exit codes stable?
- Can tests capture output without invoking real Sway?
- Are destructive actions explicit and validated?

## Cross-references
See `golang-testing`, `golang-dependency-injection`, `golang-project-layout` and `golang-continuous-integration`.
