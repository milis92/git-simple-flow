# Documentation Overhaul Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite project documentation from a single monolithic README into focused files — a workflow philosophy doc, installation guide, contributing guide, and lean example-driven README.

**Architecture:** Five files total. `docs/simple-flow.md` and `docs/installation.md` are new standalone docs. `CONTRIBUTING.md` is a new file following GitHub conventions. `README.md` is a full rewrite linking to the three above. `CHANGELOG.md` gets minor restructuring.

**Tech Stack:** Markdown, ASCII diagrams

**Spec:** `docs/superpowers/specs/2026-03-25-documentation-overhaul-design.md`

---

### Task 0: Update .gitignore to track docs

The current `.gitignore` contains `/docs/` which blocks all documentation files. The new `docs/simple-flow.md` and `docs/installation.md` must be tracked in git. Update `.gitignore` to only ignore `docs/superpowers/` (specs and plans are working documents, not shipped docs).

**Files:**
- Modify: `.gitignore`

- [ ] **Step 1: Update .gitignore**

Change `/docs/` to `/docs/superpowers/` so that `docs/simple-flow.md` and `docs/installation.md` are tracked, but specs/plans remain ignored.

In `.gitignore`, replace:
```
/docs/
```
with:
```
/docs/superpowers/
```

- [ ] **Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: track docs/ except superpowers working documents"
```

---

### Task 1: Create `docs/simple-flow.md`

The centerpiece document explaining Simple Flow as a git workflow philosophy. This file has no outbound links to other project docs, so it can be written first.

**Files:**
- Create: `docs/simple-flow.md`

**Reference material:**
- Spec sections: "1. `docs/simple-flow.md` — Workflow Philosophy" (lines 42-94)
- Current README feature bullets (lines 15-21) for selling points to weave in

- [ ] **Step 1: Write the document**

Create `docs/simple-flow.md` with these sections in order:

**Section 1 — Title and elevator pitch:**
```markdown
# Simple Flow

A git workflow that sits between [Git Flow](https://nvie.com/posts/a-successful-git-branching-model/) and [GitHub Flow](https://docs.github.com/en/get-started/using-github/github-flow). Feature branches with semver versioning built in — no release branches, no feature flags, no ceremony.
```

**Section 2 — Visual overview:**

A single ASCII diagram showing the complete happy path — a feature branch, a release tag, and a hotfix. Use the diagram convention from the spec: `●` for commits, `───` for history, `\` `/` for branch/merge, branch labels on the left.

```
main         ───●───────●───── v1.0.0 ───●───────●───── v1.1.0 ───●──── v1.1.1
                 \     /                  |        \     /          |
feature/login     ●──●                   |  feature/search  ●──●  |
                                         |                         |
                                    hotfix/crash ────●─────────────●
```

This diagram should show: two features merging to main, a release tag after each, and a hotfix branching from v1.0.0 that merges back after v1.1.0.

**Section 3 — Core Principles:**

A heading "## Core Principles" with 5 bullets:
1. **One trunk, many branches** — All work starts from `main` and merges back to `main`. No long-lived develop, staging, or release branches.
2. **Tags are releases** — A release is just a semver tag on `main`. No release branches, no cherry-picking, no parallel maintenance.
3. **Versioning is built in** — Every release gets a `v<major>.<minor>.<patch>` tag. The version lives in git, not in a file you manually bump.
4. **Hotfixes branch from tags** — When production breaks, branch from the release tag, fix, merge back to main, and tag a patch release. Simple, fast, auditable.
5. **No feature flags needed** — Branches can live a bit longer than GitHub Flow assumes. Ship when ready, not when the flag is flipped.

**Section 4 — Feature Workflow:**

A heading "## Feature Workflow" with its own ASCII diagram showing:
```
main         ───●─────────●───── v1.2.0
                 \       /
feature/foo       ●──●──●
                  ↑     ↑    ↑
               branch  work  PR merged
```

Then step-by-step:
1. Branch from main: `git sf feature start my-feature`
2. Work on the branch — commit as normal, push when ready
3. Open a PR: `git sf feature publish`
4. Merge and clean up: `git sf feature finish`
5. Optionally tag a release: `git sf release minor`

Add a brief note: feature branches can live for days or weeks. Unlike GitHub Flow, there's no pressure to merge the same day. Unlike Git Flow, there's no develop branch accumulating unreleased work.

**Section 5 — Hotfix Workflow:**

A heading "## Hotfix Workflow" with its own ASCII diagram:
```
main         ───●───── v1.0.0 ──────────●───── v1.0.1
                                |       /
hotfix/crash                    ●──●──●
                                ↑     ↑
                           branch from tag
                                   PR merged + auto-tag
```

Then step-by-step:
1. Branch from the latest release tag: `git sf hotfix start crash-fix`
2. Fix the issue, commit
3. Open a PR: `git sf hotfix publish`
4. Merge and auto-tag a patch release: `git sf hotfix finish --release`

Key point: the hotfix branches from the *tag*, not from main. This means it only contains the released code plus the fix — no unreleased features leak into the patch.

**Section 6 — Release Workflow:**

A heading "## Release Workflow" with a simple diagram:
```
main         ───●───●───●───── v1.3.0
                          ↑
                     tag + push
```

Then step-by-step:
1. Ensure main is up to date
2. Tag: `git sf release minor` (or `major` / `patch`)
3. Confirm the version when prompted
4. The tag is pushed to origin, triggering CI/CD

Note: releases don't create branches. A release is a point-in-time snapshot via a git tag.

**Section 7 — Comparison table:**

A heading "## How Simple Flow Compares" with this table:

| | Git Flow | GitHub Flow | Simple Flow |
|---|---|---|---|
| **Trunk branch** | `develop` + `main` | `main` | `main` |
| **Release branches** | Yes | No | No |
| **Feature branches** | Long-lived | Short-lived | Flexible |
| **Hotfix path** | Branch from main, merge to main + develop | Just a PR | Branch from tag, merge to main |
| **Built-in versioning** | No | No | Yes (semver tags) |
| **Feature flags needed** | Rarely | Often | No |
| **Ceremony** | High | Low | Low |
| **Best for** | Scheduled releases, multiple environments | Continuous deployment | Trunk-based with versioned releases |

**Section 8 — Edge Cases & Patterns:**

A heading "## Edge Cases" with 5 subsections (use `###` headings):

1. **Hotfix while a feature is in progress** — The hotfix takes priority. Merge it first. After the hotfix lands and is tagged, rebase or merge main into your feature branch to pick up the fix. `git sf` does not automate this — just `git rebase main` or `git merge main` on your feature branch.

2. **Multiple concurrent feature branches** — Each feature branch is independent. They merge to main separately. No coordination needed unless they touch the same files, in which case normal merge conflict resolution applies.

3. **Two releases cut close together** — Tags are sequential on main. If someone tags `v1.2.0` and you immediately try to tag another release, `git sf` will see `v1.2.0` as the latest and bump from there (e.g., `v1.3.0` for minor). There's no conflict — tags are immutable points in history.

4. **Feature branch falls behind main** — Before your PR can merge cleanly, you may need to update your branch. Rebase (`git rebase main`) or merge (`git merge main`) — your choice. `git sf` doesn't enforce a strategy here; it just needs the PR to be mergeable.

5. **Hotfix needs unreleased feature code** — It shouldn't. Always branch hotfixes from the release tag, not from main. The hotfix should contain only the released code plus the minimal fix. If the fix genuinely depends on unreleased work, that's a signal to cut a new release from main instead of a hotfix.

**Section 9 — When NOT to Use Simple Flow:**

A heading "## When to Use Something Else" with 4 items:

1. **Parallel release lines** (v1.x and v2.x maintained simultaneously) — Simple Flow assumes one version stream. Use Git Flow with release branches if you support multiple major versions.
2. **Monorepo with independent packages** — If packages release on different cadences, Simple Flow's single-version model doesn't fit. Consider per-package tooling.
3. **Continuous deployment with no versions** — If you deploy every merged commit and never cut named releases, plain GitHub Flow is simpler. Simple Flow's value is in the versioning.
4. **Feature flags required by policy** — Simple Flow's branch model makes flags unnecessary, but if organizational policy mandates them regardless, the branching model adds no value over GitHub Flow.

- [ ] **Step 2: Review the file**

Read back `docs/simple-flow.md` and verify:
- At least 3 ASCII git-tree diagrams (feature, hotfix, release + overview = 4)
- All 5 edge cases covered
- All 4 "when not to use" scenarios covered
- No broken links

- [ ] **Step 3: Commit**

```bash
git add docs/simple-flow.md
git commit -m "docs: add Simple Flow workflow philosophy guide"
```

---

### Task 2: Create `docs/installation.md`

All installation methods moved from README, plus shell completions and the git-sf naming explanation.

**Files:**
- Create: `docs/installation.md`

**Reference material:**
- Current README installation section (lines 25-70)
- Current README completion section (lines 200-211)
- Spec section: "3. `docs/installation.md`" (lines 124-133)
- `cmd/completion.go` for valid shell names: bash, zsh, fish, powershell

- [ ] **Step 1: Write the document**

Create `docs/installation.md` with these sections:

**Title:**
```markdown
# Installation
```

**Section 1 — Quick Install:**
```markdown
## Quick Install

The fastest way to get started:

**macOS / Linux (Homebrew):**
\`\`\`sh
brew install nickssmallpdf/tap/git-sf
\`\`\`

**Any platform with Go:**
\`\`\`sh
go install github.com/nickssmallpdf/git-sf@latest
\`\`\`
```

**Section 2 — Package Managers:**

A heading "## Package Managers" with subsections for each method. Copy the exact install commands from the current README (lines 27-64) — these are correct and should not change:
- Homebrew (macOS / Linux) — `brew install nickssmallpdf/tap/git-sf`
- APT (Debian / Ubuntu) — curl + dpkg
- RPM (Fedora / RHEL) — curl + rpm
- Snap — `sudo snap install git-sf --classic`
- Scoop (Windows) — bucket add + install
- Go install — `go install github.com/nickssmallpdf/git-sf@latest`

**Section 3 — Manual Download:**

```markdown
## Manual Download

Download a pre-built binary for your platform from the [Releases page](https://github.com/nickssmallpdf/git-sf/releases).

Extract the archive and place the `git-sf` binary somewhere on your `PATH`.
```

**Section 4 — Verify:**

```markdown
## Verify Installation

\`\`\`sh
git sf --help
\`\`\`

You should see the list of available commands.
```

**Section 5 — How Git Recognizes It:**

```markdown
## How It Works with Git

Git automatically discovers any executable named `git-<name>` on your `PATH` and exposes it as `git <name>`. Because the binary is called `git-sf`, you can run:

\`\`\`sh
git sf feature start my-feature
\`\`\`

No aliases or configuration needed.
```

**Section 6 — Shell Completions:**

```markdown
## Shell Completions

Generate and source a completion script for tab completion of commands and flags.

**Bash:**
\`\`\`sh
# Add to ~/.bashrc
eval "$(git sf completion bash)"
\`\`\`

**Zsh:**
\`\`\`sh
# Add to ~/.zshrc
eval "$(git sf completion zsh)"
\`\`\`

**Fish:**
\`\`\`sh
git sf completion fish | source
# To persist: git sf completion fish > ~/.config/fish/completions/git-sf.fish
\`\`\`

**PowerShell:**
\`\`\`powershell
# Add to your PowerShell profile
git sf completion powershell | Out-String | Invoke-Expression
\`\`\`
```

- [ ] **Step 2: Commit**

```bash
git add docs/installation.md
git commit -m "docs: add installation guide"
```

---

### Task 3: Create `CONTRIBUTING.md`

New file following GitHub conventions. Contains dev setup, architecture, testing, conventions, and release process — all moved from the current README's Development and Architecture sections.

**Files:**
- Create: `CONTRIBUTING.md`

**Reference material:**
- Current README Development section (lines 254-270)
- Current README Architecture table (lines 276-289)
- Spec section: "4. `CONTRIBUTING.md`" (lines 135-163)
- `go.mod` for Go version (1.26.1)
- `.goreleaser.yml` for release process details

- [ ] **Step 1: Write the document**

Create `CONTRIBUTING.md` with these sections:

**Title:**
```markdown
# Contributing to git-sf
```

**Section 1 — Prerequisites:**
```markdown
## Prerequisites

- [Go](https://go.dev/) 1.26+ (see `go.mod` for exact version)
- [git](https://git-scm.com/) 2.x or later
- [gh](https://cli.github.com/) (GitHub CLI) — needed to run integration tests
- [golangci-lint](https://golangci-lint.run/) — for linting
```

**Section 2 — Getting Started:**
```markdown
## Getting Started

\`\`\`sh
git clone https://github.com/nickssmallpdf/git-sf.git
cd git-sf
go build -o git-sf .
./git-sf --help
\`\`\`
```

**Section 3 — Project Structure:**

A heading "## Project Structure" with this table (derived from filesystem, per spec):

| Directory | Purpose |
|---|---|
| `cmd/` | Cobra command definitions (feature, hotfix, release, status, config, completion) |
| `internal/config/` | Three-layer config loading (defaults → global → repo) |
| `internal/runner/` | Command runner abstraction with dry-run and verbose support |
| `internal/git/` | Git operations wrapper (branch, tag, merge, preflight checks) |
| `internal/gh/` | GitHub CLI wrapper (PR create, merge, check status) |
| `internal/feature/` | Feature branch business logic |
| `internal/hotfix/` | Hotfix branch business logic |
| `internal/release/` | Release tagging business logic |
| `internal/status/` | Status display business logic |
| `internal/ui/` | Styled terminal output (lipgloss) |
| `internal/version/` | Semantic version parsing, bumping, comparison |
| `test/` | Integration tests (build binary, create temp repos, run workflows) |

Add a brief note: "All internal packages use interface-based design for testability. `runner.Runner` is the central abstraction for executing shell commands, supporting both real execution and dry-run mode."

**Section 4 — Testing:**
```markdown
## Testing

**Unit tests** — test internal packages in isolation:
\`\`\`sh
go test ./internal/... -v
\`\`\`

**Integration tests** — build the binary and run it against temporary git repos:
\`\`\`sh
go test ./test/... -v -count=1
\`\`\`

**All tests with coverage:**
\`\`\`sh
go test ./... -v -coverprofile=coverage.out -covermode=atomic
\`\`\`

Integration tests create real git repositories in temp directories, run actual `git` and `git-sf` commands, and verify the results. They require `git` and `gh` to be installed.
```

**Section 5 — Linting:**
```markdown
## Linting

\`\`\`sh
golangci-lint run
\`\`\`
```

**Section 6 — Conventions:**
```markdown
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
```

**Section 7 — Release Process:**
```markdown
## Release Process

Releases are automated via [GoReleaser](https://goreleaser.com/). When a tag matching `v*.*.*` is pushed, CI builds binaries for all platforms (Linux, macOS, Windows × AMD64, ARM64) and publishes to:

- GitHub Releases (tar.gz/zip archives)
- Homebrew (`nickssmallpdf/homebrew-tap`)
- Scoop (`nickssmallpdf/scoop-bucket`)
- Snap Store
- Debian and RPM packages

Contributors don't need to run GoReleaser locally. Just open a PR, get it merged, and a maintainer will tag the release.
```

- [ ] **Step 2: Commit**

```bash
git add CONTRIBUTING.md
git commit -m "docs: add contributing guide"
```

---

### Task 4: Rewrite `README.md`

Full rewrite. This is the last doc to write because it links to the three files created in Tasks 1-3.

**Files:**
- Modify: `README.md` (full overwrite)

**Reference material:**
- Spec section: "2. `README.md` — The Front Door" (lines 96-122)
- Current README for badges (line 7) and config table (lines 223-232)
- Success criteria: under 150 lines, no flag tables

- [ ] **Step 1: Write the new README**

Overwrite `README.md` with this structure:

**Hero banner** (centered div):
- `# git-sf`
- Tagline: "Simple Flow — a lightweight, opinionated Git workflow CLI for trunk-based development."
- Same three badges as current README (CI, Go version, License)

**Why section:**
```markdown
## Why git-sf?

Teams doing trunk-based development still juggle `git`, `gh`, and manual versioning across multiple commands. `git-sf` wraps the full workflow — branch, PR, merge, tag — into simple commands with [semver versioning built in](docs/simple-flow.md).

> **New to Simple Flow?** Read the [workflow guide](docs/simple-flow.md) to understand the philosophy before diving into commands.
```

**Quick Start:**
```markdown
## Quick Start

\`\`\`sh
git sf feature start my-feature   # branch from main
git sf feature publish             # push + open PR
git sf feature finish              # merge PR + clean up
git sf release minor               # tag v1.x.0
\`\`\`

See the [installation guide](docs/installation.md) for all install methods. The fastest: `brew install nickssmallpdf/tap/git-sf`
```

**Usage section:**

A heading "## Usage" followed by example-driven command descriptions. Use inline code, not tables. Format as a definition list style:

```markdown
## Usage

Every command supports `--dry-run` (preview without executing) and `--verbose` (print each command as it runs).

### Features

\`\`\`sh
git sf feature start my-feature    # create feature/my-feature from main
git sf feature publish              # push branch + open PR
git sf feature finish               # merge PR, delete branch, switch to main
git sf feature discard              # close PR, delete branch, switch to main
\`\`\`

### Hotfixes

\`\`\`sh
git sf hotfix start crash-fix      # branch from latest release tag
git sf hotfix publish               # push branch + open PR
git sf hotfix finish                # merge PR, clean up
git sf hotfix finish --release      # merge + auto-tag patch release
\`\`\`

### Releases

\`\`\`sh
git sf release minor               # tag next minor version (default)
git sf release major               # tag next major version
git sf release patch               # tag next patch version
\`\`\`

### Status & Config

\`\`\`sh
git sf status                      # branch info, PR status, latest tag
git sf config                      # show effective config + sources
git sf init                        # create .sfconfig.yml with defaults
\`\`\`

Run `git sf <command> --help` for all available flags.
```

**Configuration section:**

Keep the config table from the current README (lines 223-232) and the example `.sfconfig.yml`. Keep the 3-layer explanation. This is compact and useful inline.

**Requirements section:**
```markdown
## Requirements

- **git** 2.x or later
- **gh** ([GitHub CLI](https://cli.github.com/)) — required for PR operations
```

**License section:**
```markdown
## License

MIT. See [LICENSE](LICENSE).
```

**Links footer (centered div):**
```markdown
---

<div align="center">

[Installation](docs/installation.md) · [Simple Flow Workflow](docs/simple-flow.md) · [Contributing](CONTRIBUTING.md) · [Changelog](CHANGELOG.md)

</div>
```

- [ ] **Step 2: Verify line count**

```bash
wc -l README.md
```

Expected: under 150 lines. If over, trim the Configuration section first (it's the most compressible).

- [ ] **Step 3: Verify no flag tables**

```bash
grep -c "^|.*Flag.*|" README.md
```

Expected: 0

- [ ] **Step 4: Verify cross-references resolve**

```bash
# All linked files must exist
test -f docs/simple-flow.md && echo "OK: simple-flow.md" || echo "MISSING: simple-flow.md"
test -f docs/installation.md && echo "OK: installation.md" || echo "MISSING: installation.md"
test -f CONTRIBUTING.md && echo "OK: CONTRIBUTING.md" || echo "MISSING: CONTRIBUTING.md"
test -f CHANGELOG.md && echo "OK: CHANGELOG.md" || echo "MISSING: CHANGELOG.md"
test -f LICENSE && echo "OK: LICENSE" || echo "MISSING: LICENSE"
```

Expected: all OK

- [ ] **Step 5: Commit**

```bash
git add README.md
git commit -m "docs: rewrite README as example-driven front door"
```

---

### Task 5: Restructure `CHANGELOG.md`

Minor edits to the existing changelog. Split entries into user-facing and internal, add comparison links.

**Files:**
- Modify: `CHANGELOG.md`

**Reference material:**
- Spec section: "5. `CHANGELOG.md`" (lines 165-175)
- Current CHANGELOG.md content (40 lines)
- Classification rule from spec: test/, CI, linting config, developer tooling → Internal. Everything else → user-facing.

- [ ] **Step 1: Rewrite CHANGELOG.md**

Apply these changes:

**[Unreleased] section — reclassify entries:**

User-facing `### Added`:
- CHANGELOG.md

Internal `### Internal`:
- CLAUDE.md with project overview and developer conventions
- golangci-lint configuration with stricter linters
- Test coverage tracking with Codecov in CI
- Formatting check (`gofmt`) in CI
- Dependabot for automated dependency updates
- Integration tests for config init, config show, completion, and status commands

Internal `### Changed` (rename to `### Internal` and merge with above, or keep as separate `### Changed` under Internal):
- Pinned golangci-lint and goreleaser-action versions in CI
- CI now derives Go version from go.mod instead of hardcoding

**[0.1.0] section — reclassify entries:**

User-facing `### Added`:
- Core CLI with `git sf` subcommand structure
- `feature start/publish/finish/discard` commands for feature branch workflow
- `hotfix start/publish/finish/discard` commands for hotfix workflow
- `release [major|minor|patch]` command with semver tagging and confirmation
- `status` command showing branch, PR, checks, and release info
- `config` and `init` commands for 3-layer configuration management
- `completion` command for bash, zsh, fish, and PowerShell
- Dry-run and verbose global flags
- Styled terminal output with lipgloss
- Homebrew, Snap, Scoop, deb, and rpm packaging
- README with full command reference and installation instructions
- MIT License

Internal `### Internal`:
- Integration tests for feature, hotfix, and release flows
- CI workflow with unit tests, integration tests, and linting
- GoReleaser config for multi-platform distribution (Linux/Darwin/Windows x AMD64/ARM64)

**Add comparison links at the bottom:**
```markdown
[Unreleased]: https://github.com/nickssmallpdf/git-sf/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/nickssmallpdf/git-sf/commits/v0.1.0
```

- [ ] **Step 2: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: restructure changelog with user-facing and internal sections"
```

---

### Task 6: Final verification

Run all success criteria checks from the spec.

**Files:** None (read-only)

- [ ] **Step 1: Check README line count**

```bash
wc -l README.md
```

Expected: under 150 lines

- [ ] **Step 2: Check no flag tables in README**

```bash
grep -c "^|.*Flag.*|" README.md
```

Expected: 0

- [ ] **Step 3: Check diagram count in simple-flow.md**

```bash
grep -c "^main\|^hotfix\|^feature" docs/simple-flow.md
```

Expected: at least 3 (one per workflow diagram + overview)

- [ ] **Step 4: Check CONTRIBUTING.md has build/test/lint commands**

```bash
grep -c "go build\|go test\|golangci-lint" CONTRIBUTING.md
```

Expected: at least 4 (build + 3 test commands + lint)

- [ ] **Step 5: Check all cross-references resolve**

```bash
test -f docs/simple-flow.md && test -f docs/installation.md && test -f CONTRIBUTING.md && test -f CHANGELOG.md && test -f LICENSE && echo "All links valid" || echo "BROKEN LINKS"
```

Expected: "All links valid"

- [ ] **Step 6: Review all files one more time**

Read each file in order and verify it matches the spec:
1. `docs/simple-flow.md`
2. `docs/installation.md`
3. `CONTRIBUTING.md`
4. `README.md`
5. `CHANGELOG.md`
