# Go Project Structure Refactor

**Date:** 2026-03-24
**Status:** Approved

## Summary

Three structural improvements to align with Go best practices:

1. **Extract business logic** from `cmd/` into per-domain packages under `internal/`
2. **Clean up import aliases** by renaming `internal/exec` to `internal/runner` and removing unnecessary aliases
3. **Add integration build tag** to `test/integration_test.go`

## Fix 1: Extract business logic into per-domain packages

### Problem

`cmd/feature.go` (320 lines) and `cmd/hotfix.go` (337 lines) contain business logic, UI feedback, and flag parsing all mixed into inline Cobra handlers. `cmd/release.go` and `cmd/status.go` follow the same pattern at smaller scale.

### Design

Create four new packages under `internal/`, one per command domain. Each package has a `Service` struct holding its dependencies and methods for each subcommand.

#### `internal/feature/feature.go`

```go
type Service struct {
    Git    *git.Git
    GH     *gh.GH
    UI     *ui.UI
    Config config.Config
}

type StartOpts struct {
    DraftPR bool
    Title   string
}

type PublishOpts struct {
    Title string
    Body  string
}

type FinishOpts struct {
    Force bool
}

func (s *Service) Start(name string, opts StartOpts) error
func (s *Service) Publish(opts PublishOpts) error
func (s *Service) Finish(opts FinishOpts) error
func (s *Service) Discard(reason string) error
```

Business logic moves verbatim from `cmd/feature.go` into these methods. The `cmd/feature.go` handlers become thin wrappers: parse flags into opts, construct Service, call method.

#### `internal/hotfix/hotfix.go`

Same struct pattern. Key differences from feature:
- `Start` branches from latest tag (not main)
- `Finish` has `Release bool` in opts for auto-tagging a patch release

```go
type Service struct {
    Git    *git.Git
    GH     *gh.GH
    UI     *ui.UI
    Config config.Config
}

type StartOpts struct {
    DraftPR bool
    Title   string
}

type PublishOpts struct {
    Title string
    Body  string
}

type FinishOpts struct {
    Force   bool
    Release bool
}

func (s *Service) Start(name string, opts StartOpts) error
func (s *Service) Publish(opts PublishOpts) error
func (s *Service) Finish(opts FinishOpts) error
func (s *Service) Discard(reason string) error
```

#### `internal/release/release.go`

No GH dependency needed (releases are tag-based, not PR-based).

```go
type Service struct {
    Git    *git.Git
    UI     *ui.UI
    Config config.Config
}

func (s *Service) Release(scope string) error
```

#### `internal/status/status.go`

```go
type Service struct {
    Git    *git.Git
    GH     *gh.GH
    UI     *ui.UI
    Config config.Config
}

func (s *Service) Show() error
```

### What cmd/ files become

Each Cobra handler:
1. Parses flags
2. Constructs the domain Service with dependencies
3. Calls the single service method

Example — `cmd/feature.go` finish handler:
```go
RunE: func(cmd *cobra.Command, args []string) error {
    cfg := loadConfig()
    r := runner.NewRunner(dryRun, verbose)
    svc := &feature.Service{
        Git: git.New(r, "."), GH: gh.New(r),
        UI: ui.New(), Config: cfg,
    }
    force, _ := cmd.Flags().GetBool("force")
    return svc.Finish(feature.FinishOpts{Force: force})
}
```

### No shared interfaces or helpers

Feature, hotfix, release, and status share small patterns (preflight checks ~6 lines, branch cleanup ~12 lines) but these are small enough to duplicate. Each package is self-contained.

## Fix 2: Clean up import aliases

### Problem

Current aliases in cmd files:
- `runner "github.com/nickssmallpdf/git-sf/internal/exec"` — alias needed because `exec` collides with stdlib `os/exec`
- `gitpkg "github.com/nickssmallpdf/git-sf/internal/git"` — unnecessary alias, no collision exists

The `internal/gh/gh.go` file also aliases `internal/exec` as `runner`.

### Design

**Rename `internal/exec` to `internal/runner`:**
- The package's main type is `Runner` and its constructor is `NewRunner`
- `runner.NewRunner()` is clear and needs no alias
- Eliminates ambiguity with stdlib `os/exec`
- Update all importers: `cmd/*.go`, `internal/git/git.go`, `internal/gh/gh.go`

**Remove `gitpkg` alias:**
- No cmd file imports another package named `git`
- `git.New(r, ".")` and `git.CheckGitInstalled()` read cleanly
- Update all cmd files that use the alias

### Files affected

| File | Change |
|------|--------|
| `internal/exec/runner.go` | Move to `internal/runner/runner.go`, change `package exec` to `package runner` |
| `internal/exec/runner_test.go` | Move to `internal/runner/runner_test.go`, change package declaration |
| `internal/git/git.go` | Update import path |
| `internal/git/git_test.go` | Update import path (uses `exec.NewRunner` throughout) |
| `internal/gh/gh.go` | Update import path, remove alias |
| `cmd/feature.go` | Update imports (new paths, remove aliases) |
| `cmd/hotfix.go` | Update imports (new paths, remove aliases) |
| `cmd/release.go` | Update imports (new paths, remove aliases) |
| `cmd/status.go` | Update imports (new paths, remove aliases) |

## Fix 3: Add integration build tag

### Problem

`go test ./...` runs integration tests (which build the binary and create temp repos), making it slow for quick iteration. Currently separated by path (`go test ./internal/...` vs `go test ./test/...`) but build tags are the standard Go mechanism.

### Design

Add `//go:build integration` as the first line of `test/integration_test.go`.

Update Makefile:
- `test-integration`: add `-tags integration`
- `test-all`: add `-tags integration`
- `test` (unit only): unchanged (`go test ./internal/... -v`)

```makefile
test:
	go test ./internal/... -v

test-integration:
	go test -tags integration ./test/... -v -count=1

test-all:
	go test -tags integration ./... -v -count=1
```

Update `.github/workflows/ci.yml`:
- Integration test step (line 23) must add `-tags integration`:
  `go test -tags integration ./test/... -v -count=1`
