package gh

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	runner "github.com/nickssmallpdf/git-sf/internal/exec"
)

type GH struct {
	runner *runner.Runner
}

func New(r *runner.Runner) *GH {
	return &GH{runner: r}
}

func CheckGHInstalled() error {
	_, err := exec.LookPath("gh")
	if err != nil {
		return fmt.Errorf("gh CLI is not installed or not in PATH (install from https://cli.github.com)")
	}
	return nil
}

func (g *GH) CheckAuthenticated() error {
	_, err := g.runner.Run("gh", "auth", "status")
	if err != nil {
		return fmt.Errorf("gh is not authenticated — run 'gh auth login' first")
	}
	return nil
}

type PRInfo struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	URL    string `json:"url"`
	Draft  bool   `json:"isDraft"`
}

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

func (g *GH) MergePR(strategy string) error {
	args := []string{"pr", "merge", "--" + strategy}
	_, err := g.runner.Run("gh", args...)
	return err
}

func (g *GH) ClosePR(reason string) error {
	args := []string{"pr", "close"}
	if reason != "" {
		args = append(args, "--comment", reason)
	}
	_, err := g.runner.Run("gh", args...)
	return err
}

func (g *GH) GetCurrentPR() (*PRInfo, error) {
	out, err := g.runner.Run("gh", "pr", "view", "--json", "number,title,state,url,isDraft")
	if err != nil {
		return nil, fmt.Errorf("no PR found for current branch")
	}
	var pr PRInfo
	if err := json.Unmarshal([]byte(out), &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

type CheckStatus struct {
	Name       string
	Status     string
	Conclusion string
}

func (g *GH) GetPRChecks() ([]CheckStatus, error) {
	out, err := g.runner.Run("gh", "pr", "checks", "--json", "name,status,conclusion")
	if err != nil {
		return nil, err
	}
	var checks []CheckStatus
	if err := json.Unmarshal([]byte(out), &checks); err != nil {
		return nil, err
	}
	return checks, nil
}

func HumanizeBranchName(branch, prefix string) string {
	name := strings.TrimPrefix(branch, prefix)
	name = strings.ReplaceAll(name, "-", " ")
	name = strings.ReplaceAll(name, "_", " ")
	if len(name) > 0 {
		name = strings.ToUpper(name[:1]) + name[1:]
	}
	return name
}
