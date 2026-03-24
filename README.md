# git-sf

Simple Flow — a lightweight, opinionated Git workflow CLI for trunk-based development.

Manage feature branches, hotfixes, and semver releases with a single `git sf` command. git-sf wraps `git` and the GitHub CLI (`gh`) to enforce a consistent, linear workflow: branch from `main`, open a PR, merge back, tag a release.

---

## Installation

### Homebrew (macOS / Linux)

```sh
brew install nickssmallpdf/tap/git-sf
```

### APT (Debian / Ubuntu)

```sh
curl -fsSL https://github.com/nickssmallpdf/git-sf/releases/latest/download/git-sf_linux_amd64.deb -o git-sf.deb
sudo dpkg -i git-sf.deb
```

### RPM (Fedora / RHEL)

```sh
curl -fsSL https://github.com/nickssmallpdf/git-sf/releases/latest/download/git-sf_linux_amd64.rpm -o git-sf.rpm
sudo rpm -i git-sf.rpm
```

### Snap

```sh
sudo snap install git-sf --classic
```

### Scoop (Windows)

```sh
scoop bucket add nickssmallpdf https://github.com/nickssmallpdf/scoop-bucket
scoop install git-sf
```

### Go install

```sh
go install github.com/nickssmallpdf/git-sf@latest
```

### GitHub Releases

Download a pre-built binary for your platform from the [Releases page](https://github.com/nickssmallpdf/git-sf/releases), extract the archive, and place `git-sf` somewhere on your `PATH`.

Because the binary is named `git-sf`, Git automatically exposes it as the subcommand `git sf`.

---

## Quick Start

```sh
# Initialize a config file in your repo (optional)
git sf init

# Start a feature branch
git sf feature start my-feature

# Work, commit, then push and open a PR
git sf feature publish

# Merge the PR and clean up
git sf feature finish

# Cut a release from main
git sf release minor
```

---

## Command Reference

### Global flags

| Flag | Description |
|---|---|
| `--dry-run` | Print commands without executing them |
| `--verbose` | Print each command as it executes |

---

### `git sf init`

Create a `.sfconfig.yml` file in the repo root with default settings.

| Flag | Description |
|---|---|
| `--force` | Overwrite an existing config file |

---

### `git sf config`

Show the effective configuration, with the source of each value (default, global, or repo).

---

### `git sf status`

Show the current branch type, linked PR, CI check summary, commits ahead/behind `main`, latest tag, and next release candidates.

---

### `git sf feature`

| Subcommand | Description |
|---|---|
| `feature start <name>` | Create `feature/<name>` from the latest `main` |
| `feature publish` | Push the branch and open a PR against `main` |
| `feature finish` | Merge the PR, switch back to `main`, delete the branch |
| `feature discard` | Close the PR, delete the branch, switch back to `main` |

**`feature start` flags**

| Flag | Description |
|---|---|
| `--draft-pr` | Create a draft PR immediately after branching |
| `--title <text>` | PR title (defaults to a humanized branch name) |

**`feature publish` flags**

| Flag | Description |
|---|---|
| `--title <text>` | PR title |
| `--body <text>` | PR description |

**`feature finish` flags**

| Flag | Description |
|---|---|
| `--force` | Merge even if PR checks are failing |

**`feature discard` flags**

| Flag | Description |
|---|---|
| `--reason <text>` | Comment to post on the closed PR |

---

### `git sf hotfix`

| Subcommand | Description |
|---|---|
| `hotfix start <name>` | Create `hotfix/<name>` from the latest release tag |
| `hotfix publish` | Push the branch and open a PR against `main` |
| `hotfix finish` | Merge the PR, switch back to `main`, delete the branch |
| `hotfix discard` | Close the PR, delete the branch, switch back to `main` |

**`hotfix start` flags**

| Flag | Description |
|---|---|
| `--draft-pr` | Create a draft PR immediately |
| `--title <text>` | PR title |

**`hotfix publish` flags**

| Flag | Description |
|---|---|
| `--title <text>` | PR title |
| `--body <text>` | PR description |

**`hotfix finish` flags**

| Flag | Description |
|---|---|
| `--force` | Merge even if PR checks are failing |
| `--release` | Auto-tag a patch release after merging |

**`hotfix discard` flags**

| Flag | Description |
|---|---|
| `--reason <text>` | Comment to post on the closed PR |

---

### `git sf release [major|minor|patch]`

Tag and push a semver release from `main`. Defaults to `minor` (configurable).

Requires the local branch to be in sync with `origin/main`. Prompts for confirmation before tagging.

---

### `git sf completion`

Generate shell completion scripts.

```sh
git sf completion bash   # Bash
git sf completion zsh    # Zsh
git sf completion fish   # Fish
git sf completion ps     # PowerShell
```

Source the output in your shell profile to enable tab completion.

---

## Configuration

Configuration is merged from three layers, in order of increasing priority:

1. Built-in defaults
2. Global config: `~/.config/git-sf/config.yml`
3. Repo config: `.sfconfig.yml` at the repo root

### All options

| Key | Default | Description |
|---|---|---|
| `main_branch` | `main` | The trunk branch name |
| `tag_prefix` | `v` | Prefix for release tags (e.g. `v1.2.3`) |
| `feature_prefix` | `feature/` | Prefix for feature branches |
| `hotfix_prefix` | `hotfix/` | Prefix for hotfix branches |
| `merge_strategy` | `squash` | PR merge strategy: `squash`, `merge`, or `rebase` |
| `default_release_bump` | `minor` | Default semver bump: `major`, `minor`, or `patch` |
| `draft_pr_on_start` | `false` | Automatically create a draft PR when starting a branch |
| `hotfix_auto_release` | `false` | Automatically tag a patch release after `hotfix finish` |

### Example `.sfconfig.yml`

```yaml
main_branch: main
tag_prefix: v
feature_prefix: feature/
hotfix_prefix: hotfix/
merge_strategy: squash
default_release_bump: minor
draft_pr_on_start: false
hotfix_auto_release: false
```

Run `git sf config` to inspect the resolved values and where each one comes from.

---

## Requirements

- `git` 2.x or later
- `gh` (GitHub CLI) — required for PR operations (`feature publish`, `feature finish`, `hotfix publish`, `hotfix finish`)

---

## License

MIT. See [LICENSE](LICENSE).
