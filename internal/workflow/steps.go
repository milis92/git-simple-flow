package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/ui"
)

// FinishStepDefs returns step definitions for the common PR merge + cleanup workflow.
func FinishStepDefs(mainBranch string) []ui.StepDef {
	return []ui.StepDef{
		{Label: "Check CI"},
		{Label: "Merge PR"},
		{Label: "Switch to " + mainBranch},
		{Label: "Pull latest"},
		{Label: "Delete local branch"},
		{Label: "Delete remote branch"},
	}
}

// FinishWorkflow returns a workflow function that checks CI, merges the PR,
// switches to main, pulls, and cleans up local/remote branches.
// The returned function can be composed with additional steps by wrapping it.
func FinishWorkflow(g *git.Git, ghCli *gh.GH, branch, mainBranch, mergeStrategy string, force bool) func(context.Context, ui.StepCallbacks) error {
	return func(ctx context.Context, cb ui.StepCallbacks) error {
		ctxGit := g.WithContext(ctx)
		ctxGH := ghCli.WithContext(ctx)

		// Check CI
		cb.Start()
		if !force {
			checks, err := ctxGH.GetPRChecks()
			if err != nil {
				cb.Fail(fmt.Sprintf("could not fetch PR checks: %s", err))
				return fmt.Errorf("could not fetch PR checks: %w", err)
			}
			var failing, pending []string
			for _, c := range checks {
				if c.Conclusion == "failure" || c.Conclusion == "cancelled" {
					failing = append(failing, c.Name)
				} else if c.Status != "completed" {
					pending = append(pending, c.Name)
				}
			}
			if len(failing) > 0 {
				errMsg := fmt.Sprintf("PR checks failed: %s (use --force to override)", strings.Join(failing, ", "))
				cb.Fail(errMsg)
				return fmt.Errorf("%s", errMsg)
			}
			if len(pending) > 0 {
				errMsg := fmt.Sprintf("PR checks still running: %s (use --force to override)", strings.Join(pending, ", "))
				cb.Fail(errMsg)
				return fmt.Errorf("%s", errMsg)
			}
		}
		cb.Done()

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Merge PR
		if err := cb.Run(func() error { return ctxGH.MergePR(mergeStrategy) }); err != nil {
			return err
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Switch to main
		if err := cb.Run(func() error { return ctxGit.Checkout(mainBranch) }); err != nil {
			return err
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Pull latest
		if err := cb.Run(func() error { return ctxGit.Pull() }); err != nil {
			return err
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Delete local branch
		if err := cb.Run(func() error { return ctxGit.DeleteLocalBranch(branch) }); err != nil {
			return err
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Delete remote branch (soft fail)
		cb.RunSoftFail(func() error { return ctxGit.DeleteRemoteBranch(branch) })

		return nil
	}
}

// DiscardStepDefs returns step definitions for the common discard workflow.
func DiscardStepDefs(mainBranch string) []ui.StepDef {
	return []ui.StepDef{
		{Label: "Close PR"},
		{Label: "Switch to " + mainBranch},
		{Label: "Delete local branch"},
		{Label: "Delete remote branch"},
	}
}

// DiscardWorkflow returns a workflow function that closes the PR (soft fail),
// switches to main, and cleans up local/remote branches.
func DiscardWorkflow(g *git.Git, ghCli *gh.GH, branch, mainBranch, reason string) func(context.Context, ui.StepCallbacks) error {
	return func(ctx context.Context, cb ui.StepCallbacks) error {
		ctxGit := g.WithContext(ctx)
		ctxGH := ghCli.WithContext(ctx)

		// Close PR (soft fail — PR may not exist)
		cb.Start()
		if ghErr := gh.CheckGHInstalled(); ghErr != nil {
			cb.Fail("gh CLI not available — skipped")
		} else if authErr := ctxGH.CheckAuthenticated(); authErr != nil {
			cb.Fail("not authenticated — skipped")
		} else if err := ctxGH.ClosePR(reason); err != nil {
			cb.Fail(fmt.Sprintf("could not close PR: %s", err))
		} else {
			cb.Done()
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Switch to main
		if err := cb.Run(func() error { return ctxGit.Checkout(mainBranch) }); err != nil {
			return err
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Delete local branch
		if err := cb.Run(func() error { return ctxGit.DeleteLocalBranch(branch) }); err != nil {
			return err
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Delete remote branch (soft fail)
		cb.RunSoftFail(func() error { return ctxGit.DeleteRemoteBranch(branch) })

		return nil
	}
}
