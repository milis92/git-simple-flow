# git-sf Skill Test Scenarios

## Scenario A — Feature development (happy path)

You are working in a Git repository that has `git-sf` installed (the binary is on PATH). The repo also has a `.sfconfig.yml` config file.

The user asks: "Add a dark mode toggle to the settings page."

You should start working on this task. Show the git commands you would run to begin work, and explain your git workflow approach.

Do NOT actually run commands — just describe what you would do.

## Scenario B — Hotfix (production bug)

You are working in a Git repository that has `git-sf` installed (the binary is on PATH). The repo has release tags (latest: v2.3.1).

The user says: "The payment processing endpoint is returning 500 errors in production. Fix the null pointer in handlers/payment.go line 42."

Show the git commands you would run to begin work on this fix, and explain your git workflow. Do NOT actually run commands.

## Scenario C — Publishing and finishing

You are working in a Git repository with `git-sf` installed. You are on branch `feature/add-dark-mode` and have just committed all your changes. The user says: "I think this is ready for review. Create a PR and then merge it."

Show the git commands you would run. Do NOT actually run commands.

## Scenario D — Release

You are working in a Git repository with `git-sf` installed. You are on the `main` branch. The latest release tag is v1.4.2.

The user says: "Let's cut a new minor release."

Show the git commands you would run. Do NOT actually run commands.

## Scenario E — Error recovery (finish without PR)

You are working in a Git repository with `git-sf` installed. You are on branch `feature/refactor-auth`. You have committed your changes but never published a PR. The user says: "Merge this into main."

Show the git commands you would run. Do NOT actually run commands.

## Baseline Results (RED)

Ran all 5 scenarios as subagents WITHOUT the skill.

### Scenario A — FAIL
- Agent used `git sf feature add dark-mode-ui-toggle` (wrong: should be `git sf feature start`)
- Used `git sf feature pr` and `git sf feature merge` (wrong: should be `publish` and `finish`)

### Scenario B — PASS (with caveat)
- Correctly used `git sf hotfix start/publish/finish --release`
- Only succeeded because agent read git-sf source code

### Scenario C — FAIL
- Used `git sf feature --submit` and `git sf feature --merge` (completely wrong syntax)
- Fell back to raw `gh pr create` as alternative

### Scenario D — PASS (with caveat)
- Correctly used `git sf release minor`
- Only succeeded because agent read git-sf source code

### Scenario E — PASS (with caveat)
- Correctly identified publish → finish flow
- Only succeeded because agent read git-sf source code

### Key gaps identified
1. Command syntax not discoverable — agents guess wrong without reading source
2. Subcommand names non-obvious — agents invent `pr`/`merge`/`--submit` instead of `publish`/`finish`
3. Agents fall back to raw git/gh even when git-sf is available
4. Agents that succeed only do so by reading source code

## GREEN Results

Ran all 5 scenarios as subagents WITH the skill loaded.

### Scenario A — PASS
- Correctly used `git sf feature start add-dark-mode-toggle`
- Used raw `git add`/`git commit` for work phase (correct)
- Used `git sf feature publish` then `git sf feature finish` (correct)
- No raw git/gh fallback for lifecycle ops

### Scenario B — PASS
- Correctly identified as hotfix (production issue)
- Used `git sf hotfix start fix-payment-null-pointer`
- Used `git sf hotfix publish` and `git sf hotfix finish --release`
- Used raw git for commits (correct)

### Scenario C — PASS
- Used `git sf feature publish` then `git sf feature finish` (correct)
- Did NOT double-ask before finish
- No raw git/gh fallback

### Scenario D — PASS
- Used `git sf release minor` (correct)
- Did NOT manually tag

### Scenario E — PASS
- Recognized need to `git sf feature publish` first
- Then `git sf feature finish`
- Did NOT fall back to raw `gh pr create`

### Conclusion
All 5 scenarios pass. The skill successfully teaches the correct command syntax, lifecycle order, feature vs hotfix distinction, and error recovery — without requiring agents to read source code.
