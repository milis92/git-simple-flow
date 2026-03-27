// Package gh wraps the GitHub CLI (gh) for pull request and CI check operations.
package gh

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/milis92/git-simple-flow/internal/runner"
)

// ErrNoPR is returned when no pull request exists for the current branch.
var ErrNoPR = fmt.Errorf("no PR found for current branch")

// GH provides GitHub CLI operations. It delegates command execution to a runner.Runner.
type GH struct {
	runner *runner.Runner
}

// New creates a GH instance with the given runner.
func New(r *runner.Runner) *GH {
	return &GH{runner: r}
}

// WithContext returns a copy of GH whose commands are canceled when ctx is done.
func (g *GH) WithContext(ctx context.Context) *GH {
	return &GH{runner: g.runner.WithContext(ctx)}
}

// CheckGHInstalled verifies that the gh CLI is available in PATH.
func CheckGHInstalled() error {
	_, err := exec.LookPath("gh")
	if err != nil {
		return fmt.Errorf("gh CLI is not installed or not in PATH (install from https://cli.github.com)")
	}
	return nil
}

// CheckAuthenticated verifies that the gh CLI is logged in.
func (g *GH) CheckAuthenticated() error {
	_, err := g.runner.Run("gh", "auth", "status")
	if err != nil {
		return fmt.Errorf("gh is not authenticated — run 'gh auth login' first")
	}
	return nil
}

// PRInfo holds metadata about a GitHub pull request.
type PRInfo struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	URL    string `json:"url"`
	// Draft maps to the GitHub API field "isDraft".
	Draft bool `json:"isDraft"`
}

// CreatePR creates a pull request from the current branch to base.
// If draft is true, the PR is created as a draft.
func (g *GH) CreatePR(base, title, body string, draft bool) (*PRInfo, error) {
	args := []string{"pr", "create", "--base", base, "--title", title, "--body", body}
	if draft {
		args = append(args, "--draft")
	}
	out, err := g.runner.Run("gh", args...)
	if err != nil {
		return nil, err
	}
	return &PRInfo{URL: strings.TrimSpace(out)}, nil
}

// MergePR merges the current branch's PR using the given strategy
// ("squash", "merge", or "rebase").
func (g *GH) MergePR(strategy string) error {
	args := []string{"pr", "merge", "--" + strategy}
	_, err := g.runner.Run("gh", args...)
	return err
}

// ClosePR closes the PR associated with the given branch. If reason is
// non-empty, it is posted as a comment before closing.
func (g *GH) ClosePR(branch, reason string) error {
	args := []string{"pr", "close", branch}
	if reason != "" {
		args = append(args, "--comment", reason)
	}
	_, err := g.runner.Run("gh", args...)
	return err
}

// GetCurrentPR fetches PR metadata for the current branch.
// Returns an error if no PR exists for the branch.
func (g *GH) GetCurrentPR() (*PRInfo, error) {
	out, err := g.runner.Run("gh", "pr", "view", "--json", "number,title,state,url,isDraft")
	if err != nil {
		if isNoPRViewError(err) {
			return nil, fmt.Errorf("%w: %s", ErrNoPR, err)
		}
		return nil, err
	}
	var pr PRInfo
	if err := json.Unmarshal([]byte(out), &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

func isNoPRViewError(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no pull requests found") ||
		strings.Contains(msg, "no pull request found")
}

// CheckStatus holds the result of a single CI check on a PR.
// Fields map to the gh pr checks --json schema: name, state, bucket.
type CheckStatus struct {
	Name   string `json:"name"`
	State  string `json:"state"`  // e.g. "SUCCESS", "FAILURE", "PENDING"
	Bucket string `json:"bucket"` // e.g. "pass", "fail", "pending", "skipping", "cancel"
}

// CheckIsPending reports whether the check has not reached a terminal state yet.
func CheckIsPending(check CheckStatus) bool {
	return check.Bucket == "pending"
}

// CheckAllowsMerge reports whether a check should be treated as
// passing for merge gating purposes.
func CheckAllowsMerge(check CheckStatus) bool {
	return check.Bucket == "pass" || check.Bucket == "skipping"
}

// ClassifyChecks splits checks into merge-blocking failures and in-progress checks.
func ClassifyChecks(checks []CheckStatus) (failing, pending []string) {
	for _, check := range checks {
		switch {
		case CheckIsPending(check):
			pending = append(pending, check.Name)
		case !CheckAllowsMerge(check):
			failing = append(failing, check.Name)
		}
	}
	return failing, pending
}

// GetPRChecks fetches required CI check results for the current branch's PR.
func (g *GH) GetPRChecks() ([]CheckStatus, error) {
	out, err := g.runner.Run("gh", "pr", "checks", "--required", "--json", "name,state,bucket")
	if err != nil {
		return nil, err
	}
	var checks []CheckStatus
	if err := json.Unmarshal([]byte(out), &checks); err != nil {
		return nil, err
	}
	return checks, nil
}

// HumanizeBranchName converts a branch name into a human-readable title
// by stripping the prefix, replacing hyphens and underscores with spaces,
// and capitalizing the first letter (e.g. "feature/add-auth" becomes "Add auth").
func HumanizeBranchName(branch, prefix string) string {
	name := strings.TrimPrefix(branch, prefix)
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	return name
}
