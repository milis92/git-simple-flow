## Project Overview

git-sf (Simple Flow) is a Go CLI tool that wraps `git` and `gh` (GitHub CLI) to manage feature branches, hotfixes, and semver releases via a single `git sf` command.
One trunk (`main`), releases as tags, and composable commands for the full branch-to-release lifecycle — no release branches, no develop branch.

## Prerequisites

- Go (see `go.mod` for version)
- `git` 2.x+
- `gh` (GitHub CLI) — required for PR operations and integration tests
- `golangci-lint` — pinned via Go tool directive in `go.mod`, no separate install

## Build & Run

```sh
go build -o git-sf .
./git-sf --help
```

## Test

```sh
make test               # Unit tests: go test ./internal/... -v
make test-integration   # Integration tests: go test -tags integration ./test/... -v -count=1
make test-all           # Both
make coverage           # Unit tests with coverage report
```

Integration tests build the binary, create temporary git repos, and run real `git`/`gh` commands.

## Lint & Format

```sh
make lint               # go tool golangci-lint run
make fmt                # gofmt -w .
make fmt-check          # Check formatting without modifying
```

## Architecture

- `main.go` — Entry point, calls `cmd.Execute()`
- `cmd/` — Cobra command definitions (feature, hotfix, release, status, config, completion)
- `internal/config/` — Three-layer config loading (defaults → `~/.config/git-sf/config.yml` → `.sfconfig.yml`)
- `internal/runner/` — Command execution abstraction with dry-run/verbose/query modes
- `internal/git/` — Git operations wrapper (branch, tag, merge, preflight checks)
- `internal/gh/` — GitHub CLI wrapper (PR create, merge, checks, auth)
- `internal/feature/` — Feature branch workflow service (Start/Publish/Finish/Discard)
- `internal/hotfix/` — Hotfix branch workflow service
- `internal/release/` — Release/preview tagging and version bumping
- `internal/status/` — Branch, PR, and release status display
- `internal/version/` — Semantic version parsing, bumping, comparison
- `internal/workflow/` — Multi-step workflow orchestration with progress display
- `internal/ui/` — Terminal UI (Bubble Tea progress, huh forms, lipgloss styling)
- `test/` — Integration tests

Each workflow domain (feature, hotfix, release, status) follows a **service pattern**: a `Service` struct receives `*git.Git`, `*gh.GH`, `*ui.UI`, and `config.Config` as dependencies. The `runner.Runner` abstraction underneath supports dry-run (print-only), verbose, and query (read-only, executes even in dry-run) modes.

## Conventions

- Go module: `github.com/milis92/git-simple-flow`
- CLI framework: spf13/cobra
- Commit messages follow conventional commits: `feat:`, `fix:`, `docs:`, `test:`, `ci:`, `chore:`, `deps:`, `refactor:`, `style:`
- Branch naming: `feature/<name>`, `hotfix/<name>` (prefixes configurable)
- Tag format: `v<major>.<minor>.<patch>` (prefix configurable)

## CI Pipelines

| Workflow     | Trigger           | Purpose                                          |
|--------------|-------------------|--------------------------------------------------|
| `verify.yml` | PRs to `main`     | test → lint → snapshot build (uploads artifacts) |
| `ship.yml`   | Tag `v*.*.*`      | test → lint → full GoReleaser release            |
| `test.yml`   | Reusable workflow | `make coverage` + `make test-integration`        |
| `lint.yml`   | Reusable workflow | `make fmt-check` + `make lint`                   |
