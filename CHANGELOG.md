# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/), and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- CHANGELOG.md

### Internal
- CLAUDE.md with project overview and developer conventions
- golangci-lint configuration with stricter linters
- Test coverage tracking with Codecov in CI
- Formatting check (`gofmt`) in CI
- Dependabot for automated dependency updates
- Integration tests for config init, config show, completion, and status commands
- Pinned golangci-lint and goreleaser-action versions in CI
- CI now derives Go version from go.mod instead of hardcoding

## [0.1.0]

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
- Homebrew, Scoop, deb, and rpm packaging
- README with full command reference and installation instructions
- MIT License

### Internal
- Integration tests for feature, hotfix, and release flows
- CI workflow with unit tests, integration tests, and linting
- GoReleaser config for multi-platform distribution (Linux/Darwin/Windows x AMD64/ARM64)

[Unreleased]: https://github.com/milis92/git-simple-flow/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/milis92/git-simple-flow/commits/v0.1.0
