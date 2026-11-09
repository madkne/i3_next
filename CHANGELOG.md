# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `CODEOWNERS`, pull request and issue templates, and this `CHANGELOG.md` (STRUCT-02)
- Dependency policy recording the reason-for-each-new-dependency and license-check requirement (STRUCT-03, ⛓ SEC-10)
- Import-layering enforcement via `depguard` in `.golangci.yml` (`core <- adapters/services <- api`) (STRUCT-01)
- Desktop core skeleton (from the former `0.2.1` entry): event bus, state, process supervisor, bootstrap lifecycle, version, and shared contracts
- Daemon wiring: `i3-nextd` registers the reconnecting Sway adapter → workspace service → supervised `quickshell` and `swaybg`, honours `enabled` from `services.yaml`, and logs `ServiceStateChanged` plus every workspace/output/sway event (CORE-30)

### Changed
- Default configuration is embedded from `internal/core/config/defaults/` only; `make install` installs those files and config tests parse the embedded bytes (FIX-32)
- `registry.Register`/`MustRegister` take functional options (`registry.Name`, `registry.WithRequirement`) instead of `...any` (FIX-29)
- `bootstrap` uses the typed `ShutdownTimeouter`/`Killer` interfaces instead of reflection (FIX-29)
- `finalize-task` bumps `internal/version` only at a release; docs reference `tests/vm/vm.py` (FIX-28)

### Deprecated

### Removed
- Alias types, methods and package-level helpers in `supervisor`, `registry`, `state`, `eventbus` and `bootstrap`, plus duplicate `ProcessSpec`/`RestartPolicy` fields (FIX-29)
- `configs/defaults/`, a stale duplicate of the embedded defaults (FIX-32)

### Fixed
- Bump the minimum Go toolchain to 1.26.3 so CI uses standard-library security fixes required by `govulncheck`.

### Security
