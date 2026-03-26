# Simple Flow

[README](../README.md) · [Installation](installation.md)

Simple Flow is a Git branching model that sits between [Git Flow](https://nvie.com/posts/a-successful-git-branching-model/) and [GitHub Flow](https://docs.github.com/en/get-started/using-github/github-flow).
It keeps the single-trunk simplicity of GitHub Flow but adds structured hotfix paths that Git Flow provides — without the ceremony of
release branches, develop branches, or feature flags.
You work on feature branches, merge back to `main`, and tag a release when you are ready — with semver versioning built in.

**Contents:** [Visual Overview](#visual-overview) · [Core Principles](#core-principles) · [Workflows](#workflows) · [Comparison](#how-simple-flow-compares) · [Deployment Strategy](#deployment-strategy) · [Edge Cases](#edge-cases) · [When to Use Something Else](#when-to-use-something-else)

## Visual Overview

<p align="center">
  <img alt="Simple Flow overview — features branch from main, releases are tags, hotfixes branch from tags" src="simple-flow-diagram.svg" width="820">
</p>

All work branches from `main`. Releases are tags on `main`. Hotfixes branch from the latest release tag, merge back to `main`,
and get their own patch tag. There are no long-lived branches other than `main`.

## Core Principles

- **One trunk, many branches** — All work starts from `main` and merges back. No develop, staging, or release branches.
- **Tags are releases** — A release is a semver tag on `main`, not a branch. Nothing to maintain after the fact.
- **Versioning is built in** — Every release gets `v<major>.<minor>.<patch>`. The version lives in git tags, not in a
  file you have to bump manually.
- **Hotfixes branch from tags** — Branch from the release tag, fix, merge back to `main`, tag a patch release. The fix
  contains only released code plus the change.
- **`main` is latest, tags are stable** — The tip of `main` always contains the newest work — think of it as a rolling
  "latest" channel. Tags mark the points you have explicitly blessed as stable. This separation lets you deploy and test
  from `main` without affecting users on the tagged release.
- **No feature flags required** — Branches can live for days or weeks, so you ship when the feature is ready, not when the deploy
  pipeline demands it.

## Workflows

### Feature Workflow

<p align="center">
  <img alt="Feature workflow — start, work, finish, optional release" src="simple-flow-feature.svg" width="600">
</p>

1. **Start the branch.** This creates `feature/my-feature` from the tip of `main` and switches to it.

   ```bash
   git sf feature start my-feature
   ```

   > [!TIP]
   > Pass `--draft-pr` (or enable [`draft_pr_on_start`](../README.md#configuration) in config) to push and open a draft PR in one step.

2. **Work and commit as normal.** Nothing special here — use your usual git workflow.

   ```bash
   git add .
   git commit -m "feat: add login form"
   ```

3. **Publish the branch.** Pushes to origin and opens a pull request against `main`.

   ```bash
   git sf feature publish
   ```

4. **Finish the feature.** Merges the PR (after checks pass), switches back to `main`, and deletes the feature branch
   locally and on the remote.

   ```bash
   git sf feature finish
   ```

5. **Optionally cut a release.** If this feature is worth shipping on its own, tag it.

   ```bash
   git sf release minor
   ```

> [!TIP]
> **Changed your mind?** Run `git sf feature discard` to close the PR, delete the branch, and switch back to `main`.
>
> **Check your progress** at any time with `git sf status` to see your branch, PR, and CI state.

> [!IMPORTANT]
> **On branch lifetime:** Feature branches can live for days or weeks. Unlike GitHub Flow, there is no pressure to merge
> the same day. Unlike Git Flow, there is no develop branch where unreleased work accumulates and becomes hard to reason
> about. The branch is yours until you are done. When you merge, it goes straight to the trunk.

### Hotfix Workflow

<p align="center">
  <img alt="Hotfix workflow — branch from tag, fix, finish with patch release" src="simple-flow-hotfix.svg" width="600">
</p>

1. **Start the hotfix.** This branches from the latest release tag — not from the tip of `main`. The branch contains
   only released code.

   ```bash
   git sf hotfix start crash-fix
   ```

   > [!TIP]
   > Pass `--draft-pr` (or enable [`draft_pr_on_start`](../README.md#configuration) in config) to push and open a draft PR in one step.

2. **Fix and commit.**

   ```bash
   git add .
   git commit -m "fix: prevent nil pointer on empty input"
   ```

3. **Publish the hotfix.** Pushes and opens a PR.

   ```bash
   git sf hotfix publish
   ```

4. **Finish with a release.** Merges the PR, switches to `main`, deletes the branch, and auto-tags a patch release.
   The `--release` flag (or [`hotfix_auto_release`](../README.md#configuration) in config) handles the patch bump automatically.

   ```bash
   git sf hotfix finish --release
   ```

> [!TIP]
> **Changed your mind?** Run `git sf hotfix discard` to close the PR, delete the branch, and switch back to `main`.
>
> **Check your progress** at any time with `git sf status` to see your branch, PR, and CI state.

> [!IMPORTANT]
> **Key point:** The hotfix branches from the *tag*, not from `main`. This guarantees the hotfix contains only released
> code plus the fix — no unreleased feature work leaks in.

### Release Workflow

<p align="center">
  <img alt="Release workflow — tag a point on main" src="simple-flow-release.svg" width="400">
</p>

A release is not a branch. It is a point-in-time snapshot of `main` captured as a git tag.

1. **Tag the release.** Specify the semver bump level: `major`, `minor`, or `patch`. `git sf` verifies that your local
   `main` is in sync with origin, finds the latest tag, increments the appropriate segment, and creates the new tag.

   ```bash
   git sf release minor
   ```

2. **Confirm when prompted.** The tool shows the version bump (e.g., `v1.2.0 -> v1.3.0`) and asks for confirmation.

3. **Tag is pushed to origin.** This triggers whatever CI/CD pipeline you have wired to tag events (GoReleaser, GitHub
   Actions, etc.).

> [!NOTE]
> There are no release branches. If a release needs a fix after the fact, that is a hotfix — branch from the
> tag, fix it, and cut a patch release.

## How Simple Flow Compares

|                          | Git Flow                                      | GitHub Flow           | Simple Flow                         |
|--------------------------|-----------------------------------------------|-----------------------|-------------------------------------|
| **Trunk branch**         | `develop` + `main`                            | `main`                | `main`                              |
| **Release branches**     | Yes                                           | No                    | No                                  |
| **Feature branches**     | Long-lived                                    | Short-lived           | Flexible                            |
| **Hotfix path**          | Branch from main tag, merge to main + develop | Just a PR             | Branch from tag, merge to main      |
| **Built-in versioning**  | No                                            | No                    | Yes (semver tags)                   |
| **Feature flags needed** | Rarely                                        | Often                 | No                                  |
| **Ceremony**             | High                                          | Low                   | Low                                 |
| **Best for**             | Scheduled releases, multiple environments     | Continuous deployment | Trunk-based with versioned releases |

## Deployment Strategy

Simple Flow maps naturally to three deployment channels — **dev**, **beta**, and **production** — without extra
branches or infrastructure:

| Trigger                          | Channel                  | What it contains                     | Typical use                              |
|----------------------------------|--------------------------|--------------------------------------|------------------------------------------|
| Push to a feature/hotfix branch  | **Dev**                  | Work in progress, single feature     | Developer testing, preview environments  |
| Merge to `main`                  | **Beta / RC**            | All accepted work since the last tag | Integration testing, internal dogfooding |
| Push a semver tag (`v*.*.*`)     | **Production / Stable**  | Explicitly blessed snapshot          | End-user release                         |

### How it works

<p align="center">
  <img alt="Deployment strategy — branch push triggers dev, merge triggers beta, tag triggers production" src="simple-flow-deployment.svg" width="820">
</p>

**Dev builds** are triggered by any push to a feature or hotfix branch. These are throwaway artifacts for the developer
or reviewer — deploy to a preview environment, run on a device, share with QA. They carry no version promise.

**Beta builds** are triggered when a PR merges to `main`. At any point, the tip of `main` contains every accepted change
since the last release. Treat this as a rolling "latest" or "release candidate" channel — suitable for internal
dogfooding, staging environments, or beta testers who opt in to early builds.

**Production releases** are triggered when a semver tag is pushed. This is the only thing end users see. Because the tag
points to a specific commit on `main`, you always know exactly what code is in a production build.

> [!TIP]
> Most CI systems (GitHub Actions, GitLab CI, etc.) already distinguish between branch pushes and tag pushes in their
> trigger configuration. Simple Flow takes advantage of that: no extra branches or environment-specific config needed —
> just wire your pipeline to the three triggers above.

### Example: GitHub Actions triggers

```yaml
on:
  push:
    branches: [main, 'feature/**', 'hotfix/**']
    tags: ['v*.*.*']

jobs:
  dev:
    if: startsWith(github.ref, 'refs/heads/feature/') || startsWith(github.ref, 'refs/heads/hotfix/')
    # Build preview artifact, deploy to dev environment ...

  beta:
    if: github.ref == 'refs/heads/main'
    # Build RC artifact, deploy to staging ...

  release:
    if: startsWith(github.ref, 'refs/tags/v')
    # Build production artifact, publish to registry ...
```

### Hotfix fast path

When a hotfix merges with `--release`, the patch tag is created immediately. This means the hotfix goes from dev →
beta → production in a single `git sf hotfix finish --release`, and your CI handles each stage automatically.

## Edge Cases

### Hotfix while a feature is in progress

Hotfix takes priority. Merge the hotfix first, then rebase your feature branch onto the updated `main`. `git sf` does
not automate this part — use standard git:

```bash
git checkout feature/my-feature
git rebase main
```

### Multiple concurrent feature branches

Feature branches are independent and merge separately. If two branches touch the same files, normal merge conflict
resolution applies during the PR. There is no coordination mechanism beyond what git and your code review process
already provide.

### Two releases cut close together

Tags are sequential on `main`. `git sf` always finds the latest tag and bumps from there. If you tag `v1.4.0` and then
immediately tag again, you get `v1.5.0` (or `v1.4.1`, depending on the bump level). There is no conflict.

### Feature branch falls behind main

Rebase or merge `main` into your feature branch before the PR can be merged. `git sf` does not enforce a strategy — it
only requires the PR to be mergeable. Pick whichever approach your team prefers:

```bash
# Option A: rebase
git checkout feature/my-feature
git rebase main

# Option B: merge
git checkout feature/my-feature
git merge main
```

### Hotfix needs unreleased feature code

This scenario indicates a workflow issue. Hotfixes should always branch from the tag. If the fix genuinely depends on
work that has not been released yet, the right move is to merge that work into `main` first, cut a new release from
`main`, and skip the hotfix workflow entirely.

## When to Use Something Else

1. **Parallel release lines** (maintaining v1.x and v2.x simultaneously) — Simple Flow assumes a single version stream.
   Use Git Flow with release branches, or a custom branching model that supports multiple active release lines.

2. **Monorepo with independent packages** — Simple Flow's single-version model tags the entire repository. If your
   packages version and release independently, you need per-package tooling or a monorepo-aware release system.

3. **Continuous deployment with no versions** — If every merge to `main` goes straight to production and you never need
   to refer to a version number, plain GitHub Flow is simpler. Simple Flow's value is in the versioning; remove that and
   it is just overhead.

4. **Feature flags required by policy** — If your organization mandates feature flags for every change, the branching
   model adds no value over GitHub Flow. The whole point of longer-lived branches in Simple Flow is to avoid flags — if
   you must use flags anyway, take the simpler model.

---

<div align="center">

[README](../README.md) · [Installation](installation.md) · [Contributing](../CONTRIBUTING.md)

</div>
