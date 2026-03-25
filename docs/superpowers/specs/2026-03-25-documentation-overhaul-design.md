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
README.md                    — hero intro, quick start, usage with examples, config, links out
docs/simple-flow.md          — deep workflow philosophy with git-tree diagrams, edge cases, comparisons
docs/installation.md         — all install methods, completions, verification
CONTRIBUTING.md              — dev setup, architecture, testing, conventions
CHANGELOG.md                 — restructured with user-facing vs internal grouping
```

## File Designs

### 1. `docs/simple-flow.md` — Workflow Philosophy

The centerpiece document. Explains Simple Flow as a git workflow that sits between git-flow and GitHub flow.

**Sections:**

1. **What is Simple Flow?** — One paragraph elevator pitch. Feature branches with semver versioning built in, no release branches, no feature flags needed.

2. **Visual overview** — Git-tree ASCII diagram showing the happy path: main → feature branch → PR → merge → tag release. Shows hotfix branching from a tag.

3. **Core principles** (4-5 bullets):
   - All work happens on short-to-medium-lived branches off main
   - No release branches — tags on main are releases
   - Versioning is built into the flow, not bolted on
   - Hotfixes branch from release tags, merge back to main
   - No feature flags needed — branches can live a bit longer than GitHub flow assumes

4. **Workflows in detail** — Each with its own git-tree diagram and step-by-step:
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

6. **Edge cases & patterns** — What happens when you need to hotfix while a feature is in progress, concurrent features, version conflicts, etc.

7. **When NOT to use Simple Flow** — Honest guidance on when git-flow or GitHub flow is a better fit.

### 2. `README.md` — The Front Door

**Sections:**

1. **Hero banner** (centered) — Name, one-line tagline, CI/Go/License badges

2. **Why git-sf?** — 2-3 sentences. Pain point → solution. Link to `docs/simple-flow.md` for the full workflow philosophy.

3. **Quick Start** — 4-line example (feature start → publish → finish → release). Link to `docs/installation.md` for all install methods.

4. **Usage** — Concise, example-driven command overview. Each command gets 1-2 lines showing real usage + brief description. No flag tables — those live in `--help`. Global flags mentioned once. Covers: feature, hotfix, release, status, config, init.

5. **Configuration** — The config key table (compact, useful to keep inline). Example `.sfconfig.yml`. Brief explanation of the 3-layer merge system.

6. **Requirements** — git 2.x, gh CLI. Two bullet points.

7. **Links footer** — Installation | Simple Flow Workflow | Contributing | Changelog

**Key changes from current README:**
- Full flag tables removed — `--help` is the reference
- "Why?" section added to sell the tool
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

**Sections:**

1. **Prerequisites** — Go, git, gh, golangci-lint
2. **Getting started** — Clone, build, run
3. **Project structure** — Architecture table (moved from README)
4. **Testing** — Unit tests, integration tests, coverage. Explanation of integration test approach (builds binary, temp repos).
5. **Linting** — golangci-lint run
6. **Conventions** — Conventional commits, branch naming, interface-based design, config file locations
7. **Release process** — How GoReleaser works on tag push

### 5. `CHANGELOG.md`

- Keep the Keep a Changelog format
- Add version comparison links at the bottom (GitHub compare URLs)
- Split entries into user-facing and internal headings (e.g., `### Internal` for test/CI changes)
- Clean up `[Unreleased]` grouping

## Non-Goals

- No GitHub Wiki or separate documentation site
- No auto-generated command reference
- No changes to CLI help text or code
- No docs landing page / index file

## Success Criteria

- A new user can go from "what is this?" to running their first feature workflow by reading only the README
- A user evaluating Simple Flow can understand the workflow philosophy from `docs/simple-flow.md` without installing anything
- A contributor can set up their dev environment from CONTRIBUTING.md alone
- The README is under ~150 lines (currently ~298)
