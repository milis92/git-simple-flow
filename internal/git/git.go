// Package git wraps the git CLI for branch, tag, and sync operations.
// All commands are scoped to a specific directory via the -C flag.
package git

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/milis92/git-simple-flow/internal/version"
)

// Git provides git operations scoped to a specific working directory.
// It delegates command execution to a runner.Runner instance.
type Git struct {
	runner *runner.Runner
	dir    string
}

// New creates a Git instance that operates on the given directory.
func New(r *runner.Runner, dir string) *Git {
	return &Git{runner: r, dir: dir}
}

// WithContext returns a copy of Git whose commands are canceled when ctx is done.
func (g *Git) WithContext(ctx context.Context) *Git {
	return &Git{runner: g.runner.WithContext(ctx), dir: g.dir}
}

// ForQuery returns a copy of Git that always executes commands, even during
// dry-run mode. Use this for read-only operations like CurrentBranch.
func (g *Git) ForQuery() *Git {
	return &Git{runner: g.runner.ForQuery(), dir: g.dir}
}

// run executes a git command with -C <dir> prepended to the arguments.
func (g *Git) run(args ...string) (string, error) {
	fullArgs := append([]string{"-C", g.dir}, args...)
	return g.runner.Run("git", fullArgs...)
}

// CurrentBranch returns the name of the currently checked-out branch.
func (g *Git) CurrentBranch() (string, error) {
	return g.run("rev-parse", "--abbrev-ref", "HEAD")
}

// Checkout switches to the given branch or ref.
func (g *Git) Checkout(branch string) error {
	_, err := g.run("checkout", branch)
	return err
}

// CreateBranch creates a new branch and switches to it.
func (g *Git) CreateBranch(name string) error {
	_, err := g.run("checkout", "-b", name)
	return err
}

// Pull fetches and merges changes from the remote tracking branch.
func (g *Git) Pull() error {
	_, err := g.run("pull")
	return err
}

// Push pushes the given branch to origin and sets it as the upstream.
func (g *Git) Push(branch string) error {
	_, err := g.run("push", "-u", "origin", branch)
	return err
}

// DeleteLocalBranch force-deletes a local branch.
func (g *Git) DeleteLocalBranch(branch string) error {
	_, err := g.run("branch", "-D", branch)
	return err
}

// DeleteRemoteBranch deletes a branch from the origin remote.
func (g *Git) DeleteRemoteBranch(branch string) error {
	_, err := g.run("push", "origin", "--delete", branch)
	return err
}

// IsClean reports whether the working tree has no uncommitted changes.
func (g *Git) IsClean() (bool, error) {
	out, err := g.run("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out == "", nil
}

// Tag creates a lightweight tag with the given name at HEAD.
func (g *Git) Tag(name string) error {
	_, err := g.run("tag", name)
	return err
}

// TagAnnotated creates an annotated tag with the given message.
func (g *Git) TagAnnotated(name, message string) error {
	_, err := g.run("tag", "-a", name, "-m", message)
	return err
}

// PushTag pushes a single tag to origin.
func (g *Git) PushTag(name string) error {
	_, err := g.run("push", "origin", name)
	return err
}

// LatestTag finds the highest semver tag matching the given prefix.
// It lists all tags matching "<prefix>*", parses each as a semver version
// (skipping any that fail to parse), sorts them, and returns the tag string
// for the highest version. Returns an error if no matching tags are found.
func (g *Git) LatestTag(prefix string) (string, error) {
	out, err := g.run("tag", "-l", prefix+"*")
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", fmt.Errorf("no tags found matching %s*", prefix)
	}
	tags := strings.Split(out, "\n")
	var versions []version.Version
	tagMap := make(map[string]string)
	for _, tag := range tags {
		v, err := version.Parse(strings.TrimPrefix(tag, prefix))
		if err != nil {
			continue
		}
		versions = append(versions, v)
		tagMap[v.String()] = tag
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("no valid semver tags found matching %s*", prefix)
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].LessThan(versions[j])
	})
	latest := versions[len(versions)-1]
	return tagMap[latest.String()], nil
}

// Fetch downloads objects and refs from origin.
func (g *Git) Fetch() error {
	_, err := g.run("fetch", "origin")
	return err
}

// IsInSyncWithRemote reports whether the local branch and its origin counterpart
// point to the same commit.
func (g *Git) IsInSyncWithRemote(branch string) (bool, error) {
	local, err := g.run("rev-parse", branch)
	if err != nil {
		return false, err
	}
	remote, err := g.run("rev-parse", "origin/"+branch)
	if err != nil {
		return false, err
	}
	return local == remote, nil
}

// ListBranches returns the names of all local branches.
func (g *Git) ListBranches() ([]string, error) {
	out, err := g.run("branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	lines := strings.Split(out, "\n")
	branches := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			branches = append(branches, trimmed)
		}
	}
	return branches, nil
}

// CommitsAheadBehind returns how many commits branch is ahead of and behind base.
func (g *Git) CommitsAheadBehind(branch, base string) (ahead, behind int, err error) {
	out, err := g.run("rev-list", "--left-right", "--count", base+"..."+branch)
	if err != nil {
		return 0, 0, err
	}
	_, err = fmt.Sscanf(out, "%d\t%d", &behind, &ahead)
	return ahead, behind, err
}

// RevParse resolves a ref (branch, tag, HEAD, etc.) to its commit SHA.
func (g *Git) RevParse(ref string) (string, error) {
	return g.run("rev-parse", ref)
}

// MergeBase returns the best common ancestor commit between two refs.
func (g *Git) MergeBase(a, b string) (string, error) {
	return g.run("merge-base", a, b)
}

// ResetSoft moves HEAD to the given ref while keeping all changes staged.
func (g *Git) ResetSoft(ref string) error {
	_, err := g.run("reset", "--soft", ref)
	return err
}

// CommitWithMessage creates a commit with the given message from staged changes.
func (g *Git) CommitWithMessage(msg string) error {
	_, err := g.run("commit", "-m", msg)
	return err
}

// ForcePush force-pushes the given branch to origin, overwriting remote history.
func (g *Git) ForcePush(branch string) error {
	_, err := g.run("push", "--force", "origin", branch)
	return err
}
