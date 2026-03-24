# Contributing to git-sf

## Prerequisites

- [Go](https://go.dev/) 1.26+ (see `go.mod` for exact version)
- [git](https://git-scm.com/) 2.x or later
- [gh](https://cli.github.com/) (GitHub CLI) — needed to run integration tests
- [golangci-lint](https://golangci-lint.run/) — for linting

## Getting Started

```sh
git clone https://github.com/milis92/git-simple-flow.git
cd git-sf
go build -o git-sf .
./git-sf --help
```

## Project Structure

| Directory           | Purpose                                                                          |
|---------------------|----------------------------------------------------------------------------------|
| `cmd/`              | Cobra command definitions (feature, hotfix, release, status, config, completion) |
| `internal/config/`  | Three-layer config loading (defaults → global → repo)                            |
| `internal/runner/`  | Command runner abstraction with dry-run and verbose support                      |
| `internal/git/`     | Git operations wrapper (branch, tag, merge, preflight checks)                    |
| `internal/gh/`      | GitHub CLI wrapper (PR create, merge, check status)                              |
| `internal/feature/` | Feature branch business logic                                                    |
| `internal/hotfix/`  | Hotfix branch business logic                                                     |
| `internal/release/` | Release tagging business logic                                                   |
| `internal/status/`  | Status display business logic                                                    |
| `internal/ui/`      | Styled terminal output (lipgloss)                                                |
| `internal/version/` | Semantic version parsing, bumping, comparison                                    |
| `test/`             | Integration tests (build binary, create temp repos, run workflows)               |

> All internal packages use interface-based design for testability. `runner.Runner` is the central abstraction for
> executing shell commands, supporting both real execution and dry-run mode.

## Testing

**Unit tests** — test internal packages in isolation:

```sh
go test ./internal/... -v
```

**Integration tests** — build the binary and run it against temporary git repos:

```sh
go test ./test/... -v -count=1
```

**All tests with coverage:**

```sh
go test ./... -v -coverprofile=coverage.out -covermode=atomic
```

Integration tests create real git repositories in temp directories, run actual `git` and `git-sf` commands, and verify
the results. They require `git` and `gh` to be installed.

## Linting

```sh
golangci-lint run
```

## Conventions

**Commit messages** follow [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` — new feature
- `fix:` — bug fix
- `docs:` — documentation only
- `test:` — adding or updating tests
- `ci:` — CI/CD changes
- `chore:` — maintenance, dependencies

**Branch naming:** `feature/<name>` or `hotfix/<name>`.

**Config files:**

- Repo-level: `.sfconfig.yml` at the repo root
- Global: `~/.config/git-sf/config.yml`

## Release Process

Releases are automated via [GoReleaser](https://goreleaser.com/). When a tag matching `v*.*.*` is pushed, CI builds
binaries for all platforms (Linux, macOS, Windows × AMD64, ARM64) and publishes to:

- GitHub Releases (tar.gz/zip archives)
- Homebrew (`milis92/homebrew-tap`)
- Scoop (`milis92/scoop-bucket`)
- Snap Store
- Debian and RPM packages

Contributors don't need to run GoReleaser locally. Just open a PR, get it merged, and a maintainer will tag the release.
