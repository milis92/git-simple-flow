# Documentation Overhaul — Design Spec

## Problem

The current README.md is a single monolithic file that mixes end-user documentation, contributor guidance, full command reference tables, and installation instructions. The tone targets developers who already know the tool rather than helping new adopters understand why it exists. There is no standalone explanation of Simple Flow as a workflow philosophy.

## Goals

1. Rewrite documentation with a clear audience hierarchy: end users/adopters first, contributors second
2. Split into focused files that each serve one purpose
3. Create a deep workflow philosophy document that explains Simple Flow as a git workflow (not just a CLI tool)
4. Make the README example-driven rather than reference-driven

## Approach

Approach C — Minimal split following GitHub conventions.

## File Structure

```
README.md                    — hero intro, quick start, usage with examples, config, links out  (REWRITE)
docs/simple-flow.md          — deep workflow philosophy with git-tree diagrams, edge cases       (NEW)
docs/installation.md         — all install methods, completions, verification                    (NEW)
CONTRIBUTING.md              — dev setup, architecture, testing, conventions                      (NEW)
CHANGELOG.md                 — restructured with user-facing vs internal grouping                (EDIT)
```

Note: The existing `docs/superpowers/` directory (containing specs and plans) is unrelated and should be left untouched.

## Implementation Order

Files should be created in this order due to cross-reference dependencies:

1. `docs/simple-flow.md` — standalone, no outbound doc links
2. `docs/installation.md` — standalone, no outbound doc links
3. `CONTRIBUTING.md` — standalone, no outbound doc links
4. `README.md` — links to all three above, so must be written last
5. `CHANGELOG.md` — independent, can be done at any point

## File Designs

### 1. `docs/simple-flow.md` — Workflow Philosophy

The centerpiece document. Explains Simple Flow as a git workflow that sits between git-flow and GitHub flow.

**Sections:**

1. **What is Simple Flow?** — One paragraph elevator pitch. Feature branches with semver versioning built in, no release branches, no feature flags needed.

2. **Visual overview** — Git-tree ASCII diagram showing the happy path: main → feature branch → PR → merge → tag release. Shows hotfix branching from a tag. Diagram style uses the following convention:

   ```
   main  ───●───────●─────────●──── v1.1.0
             \     /          |
   feature/x  ●──●           |
                         hotfix/y ──●── (merged, tagged v1.1.1)
   ```

   Use `●` for commits, `───` for linear history, `\` and `/` for branch/merge points, and branch labels on the left. Each of the three workflow sections (feature, hotfix, release) gets its own diagram in this style.

3. **Core principles** (4-5 bullets):
   - All work happens on short-to-medium-lived branches off main
   - No release branches — tags on main are releases
   - Versioning is built into the flow, not bolted on
   - Hotfixes branch from release tags, merge back to main
   - No feature flags needed — branches can live a bit longer than GitHub flow assumes

4. **Workflows in detail** — Each with its own git-tree diagram (using the style above) and step-by-step:
   - Feature workflow — branch from main, work, PR, merge, tag if needed
   - Hotfix workflow — branch from release tag, fix, PR, merge, auto-patch-tag
   - Release workflow — tag main, push

5. **Comparison table** — Simple Flow vs Git Flow vs GitHub Flow:

   | Aspect | Git Flow | GitHub Flow | Simple Flow |
   |---|---|---|---|
   | Release branches | Yes | No | No |
   | Feature branches | Yes (long-lived) | Yes (short-lived) | Yes (flexible lifespan) |
   | Hotfix path | Yes (complex) | No (just a PR) | Yes (simple) |
   | Built-in versioning | No | No | Yes |
   | Feature flags needed | No | Often | No |

6. **Edge cases & patterns** — Cover these specific scenarios:
   1. Hotfix needed while a feature branch is in progress (hotfix takes priority, feature rebases after)
   2. Multiple concurrent feature branches (each merges independently, no coordination needed)
   3. Tag/version conflict when two releases are cut close together (tags are sequential on main, second release bumps from the first)
   4. Feature branch falls behind main (rebase or merge main into feature before PR)
   5. Hotfix needs to include changes from an unreleased feature (it shouldn't — hotfix from the tag, not from main)

7. **When NOT to use Simple Flow** — Cover these specific scenarios:
   1. Projects needing multiple supported release lines simultaneously (e.g., v1.x and v2.x maintained in parallel) — use git-flow with release branches
   2. Monorepos with independent release cadences per package — Simple Flow assumes one version stream
   3. Teams that deploy every commit to production automatically — GitHub flow with no tags may be simpler
   4. Projects where all features must be behind feature flags by policy — Simple Flow's branch model makes flags unnecessary, but if policy requires them, the branch model adds no value

### 2. `README.md` — The Front Door

**Sections:**

1. **Hero banner** (centered) — Name, one-line tagline, CI/Go/License badges

2. **Why git-sf?** — 2-3 sentences. Pain point: teams using trunk-based development still manually coordinate branching, PRs, and versioning across multiple tools. Solution: one command that handles the full workflow with versioning built in. Link to `docs/simple-flow.md` for the full workflow philosophy.

3. **Quick Start** — 4-line example (feature start → publish → finish → release). Link to `docs/installation.md` for all install methods.

4. **Usage** — Concise, example-driven command overview. Each command gets 1-2 lines showing real usage + brief description. No flag tables — those live in `--help`. Global flags (--dry-run, --verbose) mentioned once at the top. Covers: feature (start/publish/finish/discard), hotfix (start/publish/finish/discard), release, status, config, init. Each as a one-liner, e.g., `git sf init` — create a `.sfconfig.yml` with defaults.

5. **Configuration** — The config key table (compact, useful to keep inline). Example `.sfconfig.yml`. Brief explanation of the 3-layer merge system.

6. **Requirements** — git 2.x, gh CLI. Two bullet points.

7. **License** — One line: "MIT. See LICENSE." (preserved from current README)

8. **Links footer** — Installation | Simple Flow Workflow | Contributing | Changelog

**Key changes from current README:**
- Full flag tables removed — `--help` is the reference
- Features section removed — replaced by the more focused "Why?" section; detailed selling points live in `docs/simple-flow.md`
- "Why?" section added with clear pain-point → solution framing
- Installation section replaced by a link
- Architecture section moved to CONTRIBUTING.md
- Usage becomes example-driven, not reference-driven

### 3. `docs/installation.md`

**Sections:**

1. **Quick install** — The fastest path (Homebrew or go install)
2. **Package managers** — Homebrew, APT, RPM, Snap, Scoop, Go install (all current methods preserved)
3. **Manual download** — GitHub Releases link + instructions
4. **Verify installation** — `git sf --help`
5. **How git recognizes it** — Explanation of `git-<name>` → `git <name>` convention
6. **Shell completions** — bash/zsh/fish/powershell snippets with sourcing instructions

### 4. `CONTRIBUTING.md`

This is a **new file** (does not currently exist).

**Sections:**

1. **Prerequisites** — Go, git, gh, golangci-lint
2. **Getting started** — Clone, build, run
3. **Project structure** — Architecture table derived from the current filesystem (source of truth), not from README or CLAUDE.md. Current structure:

   | Directory | Purpose |
   |---|---|
   | `cmd/` | Cobra command definitions |
   | `internal/config/` | Three-layer config loading |
   | `internal/runner/` | Command runner abstraction with dry-run/verbose |
   | `internal/git/` | Git operations wrapper |
   | `internal/gh/` | GitHub CLI wrapper |
   | `internal/feature/` | Feature branch business logic |
   | `internal/hotfix/` | Hotfix branch business logic |
   | `internal/release/` | Release tagging business logic |
   | `internal/status/` | Status display business logic |
   | `internal/ui/` | Styled terminal output (lipgloss) |
   | `internal/version/` | Semver parsing, bumping, comparison |
   | `test/` | Integration tests |

4. **Testing** — Unit tests, integration tests, coverage. Explanation of integration test approach (builds binary, temp repos).
5. **Linting** — golangci-lint run
6. **Conventions** — Conventional commits, branch naming, interface-based design, config file locations
7. **Release process** — How GoReleaser works on tag push

### 5. `CHANGELOG.md`

- Keep the Keep a Changelog format
- Add version comparison links at the bottom using this format:
  ```
  [Unreleased]: https://github.com/nickssmallpdf/git-sf/compare/v0.1.0...HEAD
  [0.1.0]: https://github.com/nickssmallpdf/git-sf/commits/v0.1.0
  ```
  (v0.1.0 uses `/commits/` since there is no prior tag to compare against)
- Split entries into user-facing and internal headings. Classification rule: anything related to `test/`, CI workflows, linting config, or developer tooling (CLAUDE.md, golangci-lint config, Dependabot) goes under `### Internal`. Everything else (commands, features, packaging, CLI behavior) is user-facing.
- Clean up `[Unreleased]` grouping accordingly

## Non-Goals

- No GitHub Wiki or separate documentation site
- No auto-generated command reference
- No changes to CLI help text or code
- No docs landing page / index file
- No changes to CLAUDE.md (updating it to match the new architecture is a separate task)

## Success Criteria

Qualitative:
- A new user can go from "what is this?" to running their first feature workflow by reading only the README
- A user evaluating Simple Flow can understand the workflow philosophy from `docs/simple-flow.md` without installing anything
- A contributor can set up their dev environment from CONTRIBUTING.md alone

Structural (measurable):
- README is under 150 lines (currently ~298)
- README contains no flag tables
- `docs/simple-flow.md` contains at least 3 ASCII git-tree diagrams (one per workflow)
- `CONTRIBUTING.md` contains all commands needed to build, test, and lint
- All cross-references between files resolve correctly
