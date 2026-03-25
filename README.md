<div align="center">

# git-sf

Simple Git Flow — all the structure, none of the ceremony.

[![CI](https://github.com/milis92/git-simple-flow/actions/workflows/ci.yml/badge.svg)](https://github.com/milis92/git-simple-flow/actions/workflows/ci.yml) [![Go](https://img.shields.io/github/go-mod/go-version/milis92/git-simple-flow)](https://go.dev/) [![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

<picture>
  <img alt="Simple Flow — feature branches, semver releases, and hotfixes on trunk" src="docs/simple-flow-diagram.svg" width="900">
</picture>

</div>

---

## Why git-sf?

Teams still juggle `git`, `gh`, and manual versioning across multiple commands just to
branch, open a PR, merge, and tag a release. `git-sf` wraps the full workflow into simple, composable commands with
semver versioning built in — so you can stop remembering the right sequence and focus on shipping code.

> [!TIP]
> Read the [workflow guide](docs/simple-flow.md) to understand the philosophy before diving into commands.

---

## Features

- **One command per action** — start, publish, finish, or discard a branch in a single step
- **Semver releases** — tag `major`, `minor`, or `patch` releases directly from `main`
- **Hotfix from tags** — branch from the exact release, not from trunk, so no unreleased code leaks in
- **Dry-run mode** — preview every git/gh command before it runs with `--dry-run`
- **Three-layer config** — defaults, global (`~/.config/git-sf/config.yml`), and repo (`.sfconfig.yml`)
- **Shell completions** — tab completion for bash, zsh, fish, and PowerShell
- **Works as a git subcommand** — installed as `git-sf`, invoked as `git sf`

---

## Installation

> [!IMPORTANT]
> **Prerequisites:**  
> [git](https://git-scm.com/) 2.x+  
> [GitHub CLI](https://cli.github.com/) for PR operations.

**Homebrew (macOS / Linux):**

```sh
brew install milis92/tap/git-sf
```

**Go install:**

```sh
go install github.com/milis92/git-simple-flow@latest
```

Also available as `.deb`, `.rpm`, Scoop (Windows), and manual download.

**Verify it works:**

```sh
git sf --help
```

> [!TIP]
> See the [full installation guide](docs/installation.md) for all install options, configuration and shell completions.

---

## Quick Start

```sh
git sf init                        # create .sfconfig.yml with defaults
git sf feature start my-feature    # branch from main
git sf feature publish             # push + open PR
git sf feature finish              # merge PR + clean up
git sf release minor               # tag v1.x.0
```

Start by running `git sf init` to generate a `.sfconfig.yml` in your repo root with sensible defaults.
Edit it to match your team's conventions (branch prefixes, merge strategy, etc.), then follow the
branch → review → merge → release cycle above.

---

## Usage

> [!TIP]
> Every command supports `--dry-run` (preview without executing) and `--verbose` (print each command as it runs).

### Features

Features branch from `main`, get a PR, and merge back when done.

```sh
git sf feature start my-feature    # create feature/my-feature from main
git sf feature publish             # push branch + open PR
git sf feature finish              # merge PR, delete branch, switch to main
git sf feature discard             # close PR, delete branch, switch to main
```

| Flag         | Command            | Description                                     |
|--------------|--------------------|-------------------------------------------------|
| `--draft-pr` | `start`            | Create a draft PR immediately after branching   |
| `--title`    | `start`, `publish` | Override the auto-generated PR title            |
| `--body`     | `publish`          | Set the PR description                          |
| `--force`    | `finish`           | Skip CI check verification before merging       |
| `--reason`   | `discard`          | Leave a comment on the closed PR explaining why |

`start` checks out `main`, pulls latest, and creates the branch. If `--draft-pr` is passed (or `draft_pr_on_start` is
enabled in config), it also pushes and opens a draft PR in one step.

`finish` verifies all CI checks pass before merging (override with `--force`), prompts for confirmation, merges using
your configured strategy (`squash`/`merge`/`rebase`), then deletes the local and remote branches.

### Hotfixes

Hotfixes branch from the **latest release tag** — not from `main` — so unreleased code never leaks into the fix.

```sh
git sf hotfix start crash-fix      # branch from latest release tag
git sf hotfix publish              # push branch + open PR
git sf hotfix finish               # merge PR, clean up
git sf hotfix discard              # close PR, delete branch, switch to main
```

| Flag         | Command            | Description                                     |
|--------------|--------------------|-------------------------------------------------|
| `--draft-pr` | `start`            | Create a draft PR immediately after branching   |
| `--title`    | `start`, `publish` | Override the auto-generated PR title            |
| `--body`     | `publish`          | Set the PR description                          |
| `--force`    | `finish`           | Skip CI check verification before merging       |
| `--release`  | `finish`           | Auto-tag a patch release after merging          |
| `--reason`   | `discard`          | Leave a comment on the closed PR explaining why |

`--release` on `finish` (or `hotfix_auto_release` in config) automatically bumps the patch version, creates a new tag,
and pushes it — so `v1.2.3` becomes `v1.2.4` without a separate `release` command.

### Releases

Releases tag `main` with the next semver version and push the tag to origin.

```sh
git sf release minor               # tag next minor version (default)
git sf release major               # tag next major version
git sf release patch               # tag next patch version
```

The command verifies that your local `main` is in sync with origin before tagging. If no tags exist yet, the first
release starts at `v0.1.0`. The bump type argument is optional — when omitted, it uses your `default_release_bump`
config (default: `minor`).

### Status

`status` gives you a quick overview of where you are and what's going on.

```sh
git sf status
```

On a **feature or hotfix branch**, it shows:

- Branch name and type
- Associated PR number, URL, and draft status
- CI check summary (passing / failing / pending)
- How far ahead or behind `main` you are

On **main**, it shows:

- The latest release tag
- How many commits since the last release
- The next version numbers for major, minor, and patch bumps

### Config & Init

```sh
git sf init                        # create .sfconfig.yml with defaults
git sf init --force                # overwrite an existing .sfconfig.yml
git sf config                      # show effective config + sources
```

`config` displays every setting along with where it comes from — `(default)`, `(global)`, or `(repo)` — so you can see
exactly which layer is providing each value.

### Shell Completions

```sh
git sf completion bash             # output bash completions
git sf completion zsh              # output zsh completions
git sf completion fish             # output fish completions
git sf completion powershell       # output PowerShell completions
```

Pipe the output to the appropriate file for your shell. See the [installation guide](docs/installation.md) for setup
instructions.

---

## Configuration

Configuration is merged from three layers (highest priority last):

1. Built-in defaults
2. Global config — `~/.config/git-sf/config.yml`
3. Repo config — `.sfconfig.yml` at the repo root

| Key                    | Default    | Description                                       |
|------------------------|------------|---------------------------------------------------|
| `main_branch`          | `main`     | The trunk branch name                             |
| `tag_prefix`           | `v`        | Prefix for release tags (e.g. `v1.2.3`)           |
| `feature_prefix`       | `feature/` | Prefix for feature branches                       |
| `hotfix_prefix`        | `hotfix/`  | Prefix for hotfix branches                        |
| `merge_strategy`       | `squash`   | PR merge strategy: `squash`, `merge`, or `rebase` |
| `default_release_bump` | `minor`    | Default semver bump: `major`, `minor`, or `patch` |
| `draft_pr_on_start`    | `false`    | Auto-create draft PR when starting a branch       |
| `hotfix_auto_release`  | `false`    | Auto-tag patch release after `hotfix finish`      |

Example `.sfconfig.yml`:

```yaml
main_branch: main
merge_strategy: squash
default_release_bump: minor
draft_pr_on_start: true
```

Run `git sf config` to inspect the resolved values and where each one comes from.

---

## License

MIT. See [LICENSE](LICENSE).

---

<div align="center">

[Installation](docs/installation.md) · [Simple Flow Workflow](docs/simple-flow.md) · [Contributing](CONTRIBUTING.md) · [Changelog](CHANGELOG.md)

</div>
