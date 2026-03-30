# Hotfix Squash-Tag-Merge Design

Resolves [#23](https://github.com/milis92/git-simple-flow/issues/23) — Guard hotfix auto-release when main has unreleased commits.

## Problem

`git sf hotfix finish --release` currently merges the hotfix PR via GitHub, switches to `main`, pulls, and tags `main` HEAD with the next patch version. If `main` already contains unreleased feature commits, the hotfix auto-release accidentally publishes those commits too.

## Solution

Replace the "merge then tag main" flow with a **Squash-Tag-Merge** sequence when `--release` is active. The tag is created on the hotfix branch (which contains only released code plus the fix), then the branch is merged into `main` via a merge commit. The tag becomes a direct ancestor of `main` through the merge, but never includes unreleased feature work.

### Hotfix finish --release flow

```
1. Preflight       — clean tree, on hotfix branch, PR exists
2. Check CI        — sanity check before squash (same code, gh pr merge is the real gate)
3. Confirm         — user confirms merge
4. Squash          — git merge-base main HEAD → git reset --soft <base> → git commit -m "hotfix: <title>"
5. Force push      — git push --force origin <branch> (updates PR to single commit)
6. Compute version — LatestTag() → Parse() → Bump("patch")
7. Tag             — git tag <new-tag>
8. Push tag        — git push origin <new-tag> (triggers CI/CD deploy)
9. Merge PR        — gh pr merge --merge (forces --merge regardless of configured strategy)
10. Cleanup        — checkout main, pull, delete local branch, delete remote branch (soft fail)
```

### Hotfix finish without --release

Unchanged. Uses the configured merge strategy, no squash, no tag.

### Git graph result

```
(v1.2.3) A --- B --- C --- M (main)     ← "Merge hotfix v1.2.4 into main"
                \           /
                 `-- D (v1.2.4)'         ← single squashed hotfix commit
```

`git log --first-parent` shows a clean linear history. The tag `v1.2.4` is reachable from `main` via the merge commit and contains only released code plus the fix.

## Implementation

### New git primitives (`internal/git/git.go`)

Four new methods following the existing thin-wrapper pattern:

| Method | Git command | Purpose |
|--------|-------------|---------|
| `MergeBase(a, b string) (string, error)` | `git merge-base <a> <b>` | Find common ancestor of hotfix branch and main |
| `ResetSoft(ref string) error` | `git reset --soft <ref>` | Collapse commits, keep changes staged |
| `CommitWithMessage(msg string) error` | `git commit -m <msg>` | Create the squashed commit |
| `ForcePush(branch string) error` | `git push --force origin <branch>` | Update remote with squashed history |

These respect dry-run mode like all existing methods.

### Extended GH method (`internal/gh/gh.go`)

New method alongside existing `MergePR`:

```go
func (g *GH) MergePRWithMessage(strategy, subject, body string) error
```

Used by hotfix finish --release to set the merge commit subject to `"Merge hotfix <tag>"` (e.g., `"Merge hotfix v1.2.4"`). The body is left empty.

### Merge strategy override

When hotfix finish has `--release` active, the code bypasses `config.MergeStrategy` and passes `"merge"` directly to `MergePRWithMessage()`. This is necessary because `--squash` and `--rebase` create new commits on `main` that break the tag's ancestry relationship.

The `merge_strategy` config field remains — features and non-release hotfix finishes use it as before.

### Squash commit message format

`"hotfix: <PR title>"` — derived from the PR metadata already fetched in preflight. Example: `"hotfix: Fix nil pointer on empty input"`.

### Modified hotfix service (`internal/hotfix/hotfix.go`)

Both `finishInteractive` and `finishClassic` gain the new step sequence when `--release` is active. The interactive progress step definitions become:

```
Check CI → Squash commits → Force push → Create patch tag → Push tag → Merge PR → Switch to main → Pull latest → Delete local branch → Delete remote branch
```

### CI timing

CI is checked before the squash as a sanity check — same code, different commit structure. After force push, old CI results are stale, but `gh pr merge` enforces branch protection as the real gate. If branch protection requires CI to pass on the new commit, the merge will block until CI completes.

## Documentation updates

All files that describe hotfix behavior, merge strategy, or the branching model need updating:

| File | Changes |
|------|---------|
| `docs/simple-flow.md` | Hotfix workflow section (step 4 description), deployment strategy (hotfix fast path), edge cases. Update the explanation of what `--release` does. |
| `README.md` | Hotfix section: update `--release` flag description and the note about merge strategy behavior for hotfix releases. |
| `git-sf-skill/skills/git-sf-workflow/SKILL.md` | Update hotfix finish description to reflect squash-tag-merge. Note that `--release` forces `--merge` strategy. |
| `.claude-plugin/plugin/skills/git-sf-workflow/SKILL.md` | Keep in sync with the skill above. |
| `.claude/CLAUDE.md` | Update conventions section if it references hotfix tagging on main. |
| `CONTRIBUTING.md` | Update if it describes hotfix workflow for contributors. |
| `docs/installation.md` | No changes needed (does not describe hotfix behavior). |
| `.claude-plugin/plugin/tests/scenarios.md` | Update Scenario B expected behavior to reflect squash-tag-merge. |
| `docs/superpowers/specs/2026-03-25-git-sf-skill-design.md` | Update hotfix decision logic if it describes tagging behavior. |
| SVG diagrams (`simple-flow-hotfix.svg`, `simple-flow-deployment.svg`) | Update if they show the tag on main instead of the hotfix branch. |

### Key documentation points

- Hotfix `--release` squashes the branch to a single commit, tags it, then merges into main via `--merge`
- The tag lives on the hotfix branch, not on main HEAD — it only contains released code plus the fix
- Force push happens automatically during release (updates the PR to a single commit)
- `git log --first-parent` gives a clean linear history
- The tag is a direct ancestor of main (reachable via the merge commit)
- Without `--release`, hotfix finish behaves as before (configured merge strategy, no squash, no tag)
- Features are unaffected — they use the configured merge strategy

## Testing

### Unit tests (`internal/hotfix/`)

- Hotfix finish --release: verify squash → force push → tag → push tag → merge sequence
- Hotfix finish without --release: verify unchanged behavior (no squash, configured strategy)
- Verify merge strategy is forced to `"merge"` when --release is active
- Verify squash commit message format

### Unit tests (`internal/git/`)

- `MergeBase`: returns correct ancestor SHA
- `ResetSoft`: runs `git reset --soft <ref>`
- `CommitWithMessage`: runs `git commit -m <msg>`
- `ForcePush`: runs `git push --force origin <branch>`

### Integration tests (`test/`)

- End-to-end hotfix finish --release with local bare remote: verify tag exists on the squashed commit, merge commit exists on main, tag is reachable from main
- Verify `git log --first-parent main` shows the merge commit but not the squashed hotfix commit directly
