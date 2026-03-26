// Package workflow provides shared workflow steps used by both feature and
// hotfix branch lifecycles.
package workflow

import (
	"errors"
	"fmt"

	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/ui"
)

// ResolvePRInput resolves PR title and body, prompting interactively if needed.
// The prefix is stripped from the branch name to generate a default title.
func ResolvePRInput(
	u *ui.UI,
	runPrompt func(string, bool) (ui.InputPromptResult, error),
	branch, prefix, title, body string,
	includeBody bool,
) (string, string, error) {
	if u.ShouldPrompt() && (title == "" || (includeBody && body == "")) {
		defaultTitle := title
		if defaultTitle == "" {
			defaultTitle = gh.HumanizeBranchName(branch, prefix)
		}

		result, err := runPrompt(defaultTitle, includeBody)
		if err != nil {
			return "", "", err
		}
		title = result.Title
		if includeBody && body == "" {
			body = result.Body
		}
	}

	if title == "" {
		title = gh.HumanizeBranchName(branch, prefix)
	}

	return title, body, nil
}

// CurrentPRError wraps gh.ErrNoPR with a user-friendly message directing
// the user to run the given publish command first.
func CurrentPRError(err error, publishCmd string) error {
	if errors.Is(err, gh.ErrNoPR) {
		return fmt.Errorf("no PR found for this branch. Run '%s' first", publishCmd)
	}
	return err
}
