package feature

import (
	"bytes"
	"context"
	"testing"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/milis92/git-simple-flow/internal/ui"
)

func TestFinishInteractiveDryRunUsesRealRepoState(t *testing.T) {
	repoDir := initFeatureRepo(t)
	installChecksGH(t, `[{"name":"ci","state":"SUCCESS","bucket":"pass"}]`)

	var progressTitle, progressBranch string

	r := runner.NewRunner(true, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.Interactive = true
	u.AutoConfirm = true

	svc := &Service{
		Git:            git.New(r, repoDir),
		GH:             gh.New(r),
		UI:             u,
		Config:         config.Defaults(),
		RunTitlePrompt: ui.RunTitlePrompt,
		RunProgress: func(title, subtitle string, defs []ui.StepDef, workflow func(context.Context, ui.StepCallbacks) error) error {
			progressTitle = title
			progressBranch = subtitle
			return workflow(context.Background(), ui.StepCallbacks{
				Start: func() {},
				Done:  func() {},
				Fail:  func(string) {},
				Skip:  func(string) {},
			})
		},
	}

	if err := svc.Finish(FinishOpts{}); err != nil {
		t.Fatalf("Finish() error = %v, want dry-run preview to succeed", err)
	}
	if progressTitle != "git sf feature finish" {
		t.Fatalf("progress title = %q, want %q", progressTitle, "git sf feature finish")
	}
	if progressBranch != "feature/test" {
		t.Fatalf("progress branch = %q, want %q", progressBranch, "feature/test")
	}
}

func TestDiscardInteractiveDryRunUsesRealRepoState(t *testing.T) {
	repoDir := initFeatureRepo(t)

	var progressTitle, progressBranch string

	r := runner.NewRunner(true, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.Interactive = true
	u.AutoConfirm = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
		RunProgress: func(title, subtitle string, defs []ui.StepDef, workflow func(context.Context, ui.StepCallbacks) error) error {
			progressTitle = title
			progressBranch = subtitle
			return nil
		},
	}

	if err := svc.Discard(""); err != nil {
		t.Fatalf("Discard() error = %v, want dry-run preview to succeed", err)
	}
	if progressTitle != "git sf feature discard" {
		t.Fatalf("progress title = %q, want %q", progressTitle, "git sf feature discard")
	}
	if progressBranch != "feature/test" {
		t.Fatalf("progress branch = %q, want %q", progressBranch, "feature/test")
	}
}
