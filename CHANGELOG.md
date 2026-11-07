# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `CODEOWNERS`, pull request and issue templates, and this `CHANGELOG.md` (STRUCT-02)
- Dependency policy recording the reason-for-each-new-dependency and license-check requirement (STRUCT-03, ⛓ SEC-10)
- Import-layering enforcement via `depguard` in `.golangci.yml` (`core <- adapters/services <- api`) (STRUCT-01)

### Changed

### Deprecated

### Removed

### Fixed

### Security

## [0.2.1] - unreleased

Desktop core skeleton: event bus, state, process supervisor, bootstrap lifecycle, version, and shared contracts.
