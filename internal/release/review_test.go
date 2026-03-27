package release

import (
	"bytes"
	"strings"
	"testing"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/milis92/git-simple-flow/internal/ui"
)

func TestReleaseDryRunUsesRealRepoState(t *testing.T) {
	repoDir := initReleaseRepo(t)

	var out bytes.Buffer
	r := runner.NewRunner(true, false)
	u := ui.New()
	u.Out = &out
	u.AutoConfirm = true

	svc := &Service{
		Git:              git.New(r, repoDir),
		UI:               u,
		Config:           config.Defaults(),
		RunMessagePrompt: ui.RunMessagePrompt,
	}

	if err := svc.Release("patch", ""); err != nil {
		t.Fatalf("Release() error = %v, want dry-run preview to succeed", err)
	}

	if !strings.Contains(out.String(), "Next:    v0.1.1 (patch)") {
		t.Fatalf("Release() output = %q, want dry-run preview to resolve next tag from the real repo", out.String())
	}
}
