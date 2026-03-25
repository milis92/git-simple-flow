# CLAUDE.md

## Project Overview

git-sf (Simple Flow) is a lightweight CLI tool that enforces trunk-based Git workflow. It wraps `git` and `gh` (GitHub CLI) to manage feature branches, hotfixes, and semver releases via a single `git sf` command.

## Prerequisites

- Go (see `go.mod` for version)
- `git` 2.x or later
- `gh` (GitHub CLI) — required for PR operations
- `golangci-lint` — pinned via `go tool` in `go.mod`, no separate install needed

## Build & Run

    go build -o git-sf .
    ./git-sf --help

## Test

    # Unit tests
    go test ./internal/... -v

    # Integration tests (builds binary, creates temp git repos)
    go test -tags integration ./test/... -v -count=1

    # All tests with coverage
    make coverage

## Lint

    make lint

## Project Structure

- `main.go` — Entry point, calls `cmd.Execute()`
- `cmd/` — Cobra command definitions (feature, hotfix, release, status, config, completion)
- `internal/config/` — 3-layer config loading (defaults -> global -> repo)
- `internal/runner/` — Command runner abstraction with dry-run/verbose support
- `internal/git/` — Git operations wrapper (branch, tag, merge, preflight checks)
- `internal/gh/` — GitHub CLI wrapper (PR create, merge, checks)
- `internal/feature/` — Feature branch workflow (start, finish)
- `internal/hotfix/` — Hotfix branch workflow (start, finish)
- `internal/release/` — Release workflow (tag creation, version bumping)
- `internal/status/` — Repository status display (branch, PR, release info)
- `internal/ui/` — Styled terminal output using lipgloss
- `internal/version/` — Semantic version parsing, bumping, comparison
- `test/` — Integration tests that build the binary and run against temp repos

## Conventions

- Go module: `github.com/milis92/git-simple-flow`
- CLI framework: spf13/cobra + spf13/viper
- All internal packages use interface-based design for testability
- `runner.Runner` is the abstraction for running shell commands (supports dry-run)
- Config files: `.sfconfig.yml` (repo-level), `~/.config/git-sf/config.yml` (global)
- Branch prefixes: `feature/`, `hotfix/` (configurable)
- Tag format: `v<major>.<minor>.<patch>` (prefix configurable)
- Commit messages follow conventional commits: `feat:`, `fix:`, `docs:`, `test:`, `ci:`, `chore:`

## Release

Releases are automated via GoReleaser on tag push (`v*.*.*`). Multi-platform: Linux/Darwin/Windows x AMD64/ARM64.
