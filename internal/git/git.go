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

// TagAt creates a lightweight tag at a specific commit (not necessarily HEAD).
func (g *Git) TagAt(name, ref string) error {
	_, err := g.run("tag", name, ref)
	return err
}

// TagAnnotated creates an annotated tag with the given message.
func (g *Git) TagAnnotated(name, message string) error {
	_, err := g.run("tag", "-a", name, "-m", message)
	return err
}

// DeleteLocalTag removes a tag from the local repository.
func (g *Git) DeleteLocalTag(name string) error {
	_, err := g.run("tag", "-d", name)
	return err
}

// ReplaceTagAnnotated atomically replaces an existing tag with an annotated
// tag bearing the given message. If the command fails the original tag is
// preserved because git applies -f only on success.
func (g *Git) ReplaceTagAnnotated(name, message string) error {
	_, err := g.run("tag", "-a", "-f", name, "-m", message)
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
		if v.IsPrerelease() {
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

// LatestTagOnBranch finds the highest semver tag matching the given prefix
// that is reachable from ref (i.e. an ancestor of ref). This prevents
// off-branch tags (e.g. an unmerged hotfix tag) from being picked up.
func (g *Git) LatestTagOnBranch(prefix, ref string) (string, error) {
	out, err := g.run("tag", "-l", prefix+"*", "--merged", ref)
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", fmt.Errorf("no tags found matching %s* reachable from %s", prefix, ref)
	}
	tags := strings.Split(out, "\n")
	var versions []version.Version
	tagMap := make(map[string]string)
	for _, tag := range tags {
		v, err := version.Parse(strings.TrimPrefix(tag, prefix))
		if err != nil {
			continue
		}
		if v.IsPrerelease() {
			continue
		}
		versions = append(versions, v)
		tagMap[v.String()] = tag
	}
	if len(versions) == 0 {
		return "", fmt.Errorf("no valid semver tags found matching %s* reachable from %s", prefix, ref)
	}
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].LessThan(versions[j])
	})
	latest := versions[len(versions)-1]
	return tagMap[latest.String()], nil
}

// LatestPreviewTag finds the preview tag with the highest counter matching
// the given suffix and target base version, reachable from ref. Returns
// empty string (not error) if no matching preview tags exist.
func (g *Git) LatestPreviewTag(prefix, suffix, ref string, target version.Version) (string, error) {
	pattern := prefix + "*-" + suffix + ".*"
	out, err := g.run("tag", "-l", pattern, "--merged", ref)
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", nil
	}

	var bestCounter int
	var bestTag string

	for _, tag := range strings.Split(out, "\n") {
		v, err := version.Parse(strings.TrimPrefix(tag, prefix))
		if err != nil {
			continue
		}
		if v.Prerelease != suffix {
			continue
		}
		if v.Major != target.Major || v.Minor != target.Minor || v.Patch != target.Patch {
			continue
		}
		if bestTag == "" || v.PreBuild > bestCounter {
			bestCounter = v.PreBuild
			bestTag = tag
		}
	}

	return bestTag, nil
}

// ListTags returns all tags matching the given pattern.
func (g *Git) ListTags(pattern string) ([]string, error) {
	out, err := g.run("tag", "-l", pattern)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// TagExistsOnRemote checks whether a tag has been pushed to origin.
// Returns false for local-only tags that were never published.
func (g *Git) TagExistsOnRemote(tag string) (bool, error) {
	out, err := g.run("ls-remote", "--tags", "origin", tag)
	if err != nil {
		return false, err
	}
	return out != "", nil
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

// CommitCount returns the number of commits reachable from head but not from base.
func (g *Git) CommitCount(base, head string) (int, error) {
	out, err := g.run("rev-list", "--count", base+".."+head)
	if err != nil {
		return 0, err
	}
	var n int
	if _, err := fmt.Sscanf(out, "%d", &n); err != nil {
		return 0, fmt.Errorf("could not parse commit count: %w", err)
	}
	return n, nil
}

// HasCherryPickedCommits checks whether any commits in base..head have
// equivalent patches (by git patch-id) on the upstream branch. Returns true
// if cherry-picked commits are detected. Uses "git cherry" which compares
// symmetric patch-ids: a "-" prefix means the commit exists on upstream.
func (g *Git) HasCherryPickedCommits(upstream, head, base string) (bool, error) {
	out, err := g.run("cherry", upstream, head, base)
	if err != nil {
		return false, err
	}
	if out == "" {
		return false, nil
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "- ") {
			return true, nil
		}
	}
	return false, nil
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
// Uses --force-with-lease to reject the push if the remote ref has changed since
// the last fetch, preventing silent overwrites of collaborator work.
func (g *Git) ForcePush(branch string) error {
	_, err := g.run("push", "--force-with-lease", "origin", branch)
	return err
}
