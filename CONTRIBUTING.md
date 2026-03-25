# Contributing to git-sf

## Prerequisites

- [Go](https://go.dev/) 1.26+ (see `go.mod` for exact version)
- [git](https://git-scm.com/) 2.x or later
- [gh](https://cli.github.com/) (GitHub CLI) — needed to run integration tests

`golangci-lint` is managed automatically via Go's [tool directive](https://go.dev/doc/modules/managing-dependencies#tools) in `go.mod` — no separate install needed.

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
make test
# or: go test ./internal/... -v
```

**Integration tests** — build the binary and run it against temporary git repos:

```sh
make test-integration
# or: go test -tags integration ./test/... -v -count=1
```

**Unit tests with coverage:**

```sh
make coverage
```

**All tests (unit + integration):**

```sh
make test-all
```

Integration tests create real git repositories in temp directories, run actual `git` and `git-sf` commands, and verify
the results. They require `git` and `gh` to be installed.

## Linting

```sh
make lint
# or: go tool golangci-lint run
```

Format checking:

```sh
make fmt-check
```

Auto-format:

```sh
make fmt
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

Releases are automated via [GoReleaser](https://goreleaser.com/):

- **Stable releases** — when a tag matching `v*.*.*` is pushed, CI runs tests and lint, then builds binaries for all platforms (Linux, macOS, Windows × AMD64, ARM64) and publishes to GitHub Releases, Homebrew (`milis92/homebrew-tap`), Scoop (`milis92/scoop-bucket`), Snap Store, and Debian/RPM packages.

- **Latest (unstable)** — every push to `main` produces a rolling "latest" pre-release with snapshot builds (tar.gz/zip archives only, no system packages).

Contributors don't need to run GoReleaser locally. Just open a PR, get it merged, and a maintainer will tag the release.
