package git

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nickssmallpdf/git-sf/internal/runner"
	"github.com/nickssmallpdf/git-sf/internal/version"
)

type Git struct {
	runner *runner.Runner
	dir    string
}

func New(r *runner.Runner, dir string) *Git {
	return &Git{runner: r, dir: dir}
}

func (g *Git) run(args ...string) (string, error) {
	fullArgs := append([]string{"-C", g.dir}, args...)
	return g.runner.Run("git", fullArgs...)
}

func (g *Git) CurrentBranch() (string, error) {
	return g.run("rev-parse", "--abbrev-ref", "HEAD")
}

func (g *Git) Checkout(branch string) error {
	_, err := g.run("checkout", branch)
	return err
}

func (g *Git) CreateBranch(name string) error {
	_, err := g.run("checkout", "-b", name)
	return err
}

func (g *Git) Pull() error {
	_, err := g.run("pull")
	return err
}

func (g *Git) Push(branch string) error {
	_, err := g.run("push", "-u", "origin", branch)
	return err
}

func (g *Git) DeleteLocalBranch(branch string) error {
	_, err := g.run("branch", "-D", branch)
	return err
}

func (g *Git) DeleteRemoteBranch(branch string) error {
	_, err := g.run("push", "origin", "--delete", branch)
	return err
}

func (g *Git) IsClean() (bool, error) {
	out, err := g.run("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out == "", nil
}

func (g *Git) Tag(name string) error {
	_, err := g.run("tag", name)
	return err
}

func (g *Git) PushTag(name string) error {
	_, err := g.run("push", "origin", name)
	return err
}

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

func (g *Git) Fetch() error {
	_, err := g.run("fetch", "origin")
	return err
}

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

func (g *Git) CommitsAheadBehind(branch, base string) (ahead, behind int, err error) {
	out, err := g.run("rev-list", "--left-right", "--count", base+"..."+branch)
	if err != nil {
		return 0, 0, err
	}
	_, err = fmt.Sscanf(out, "%d\t%d", &behind, &ahead)
	return ahead, behind, err
}
