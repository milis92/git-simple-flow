package git

import (
	"fmt"
	"os/exec"
)

// CheckGitInstalled verifies that git is available in PATH.
func CheckGitInstalled() error {
	_, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git is not installed or not in PATH")
	}
	return nil
}

// CheckIsRepo verifies that the working directory is inside a git repository.
func (g *Git) CheckIsRepo() error {
	_, err := g.run("rev-parse", "--is-inside-work-tree")
	if err != nil {
		return fmt.Errorf("not inside a git repository")
	}
	return nil
}

// CheckCleanTree returns an error if the working tree has uncommitted changes.
func (g *Git) CheckCleanTree() error {
	clean, err := g.IsClean()
	if err != nil {
		return err
	}
	if !clean {
		return fmt.Errorf("working tree is not clean — commit or stash your changes first")
	}
	return nil
}

// CheckOnBranch returns an error if the current branch is not the expected one.
func (g *Git) CheckOnBranch(expected string) error {
	branch, err := g.CurrentBranch()
	if err != nil {
		return err
	}
	if branch != expected {
		return fmt.Errorf("must be on %s branch (currently on %s)", expected, branch)
	}
	return nil
}
