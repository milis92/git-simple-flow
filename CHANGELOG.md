# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

## [1.0.0] - 2026-03-25

### Added
- Core CLI with `git sf` subcommand structure
- `feature start/publish/finish/discard` commands for feature branch workflow
- `hotfix start/publish/finish/discard` commands for hotfix workflow
- `release [major|minor|patch]` command with semver tagging and confirmation
- `status` command showing branch, PR, checks, and release info
- `config` and `init` commands for 3-layer configuration management
- `completion` command for bash, zsh, fish, and PowerShell
- Dry-run and verbose global flags
- Styled terminal output with lipgloss
- deb and rpm packaging
- Claude Code plugin skill for guided git-sf workflows
- README with full command reference and installation instructions
- MIT License

### Fixed
- Propagate stdin read errors from `UI.Confirm()`
- Show muted warnings when status info is unavailable
- Remove snapcrafts from release config (snapcraft not available in CI)

### Internal
- Integration tests for feature, hotfix, release, config, status, and completion
- CI pipelines: PR checks, rolling latest pre-release, stable release on tag
- GoReleaser config for multi-platform distribution (Linux/Darwin/Windows x AMD64/ARM64)
- CLAUDE.md with project overview and developer conventions
- golangci-lint configuration with stricter linters
- Test coverage tracking with Codecov
- Formatting check (`gofmt`) in CI
- Dependabot for automated dependency updates
- Pinned tool versions in CI; Go version derived from go.mod

[Unreleased]: https://github.com/milis92/git-simple-flow/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/milis92/git-simple-flow/commits/v1.0.0
