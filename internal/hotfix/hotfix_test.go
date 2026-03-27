package hotfix

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/milis92/git-simple-flow/internal/ui"
)

func TestPublishPromptsBeforePush(t *testing.T) {
	repoDir := initHotfixRepo(t)
	installFakeGH(t)

	promptErr := errors.New("prompt cancelled")

	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.Interactive = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
		RunTitlePrompt: func(defaultTitle string, includeBody bool) (ui.InputPromptResult, error) {
			return ui.InputPromptResult{}, promptErr
		},
		RunProgress: ui.RunProgress,
	}

	err := svc.Publish(PublishOpts{})
	if !errors.Is(err, promptErr) {
		t.Fatalf("Publish() error = %v, want %v", err, promptErr)
	}
}

func TestPublishPromptsForBodyWhenTitleProvided(t *testing.T) {
	repoDir := initHotfixRepo(t)
	installFakeGH(t)

	promptErr := errors.New("prompt cancelled")

	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.Interactive = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
		RunTitlePrompt: func(defaultTitle string, includeBody bool) (ui.InputPromptResult, error) {
			if defaultTitle != "Already set" {
				t.Fatalf("defaultTitle = %q, want %q", defaultTitle, "Already set")
			}
			if !includeBody {
				t.Fatal("includeBody = false, want true")
			}

			return ui.InputPromptResult{}, promptErr
		},
		RunProgress: ui.RunProgress,
	}

	err := svc.Publish(PublishOpts{Title: "Already set"})
	if !errors.Is(err, promptErr) {
		t.Fatalf("Publish() error = %v, want %v", err, promptErr)
	}
}

func TestPublishSkipsOptionalPromptWhenAutoConfirm(t *testing.T) {
	repoDir := initHotfixRepo(t)
	installFakeGH(t)

	promptErr := errors.New("prompt should be skipped")

	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.Interactive = true
	u.AutoConfirm = true

	svc := &Service{
		Git:    git.New(r, repoDir),
		GH:     gh.New(r),
		UI:     u,
		Config: config.Defaults(),
		RunTitlePrompt: func(defaultTitle string, includeBody bool) (ui.InputPromptResult, error) {
			return ui.InputPromptResult{}, promptErr
		},
		RunProgress: ui.RunProgress,
	}

	err := svc.Publish(PublishOpts{})
	if err == nil {
		t.Fatal("Publish() error = nil, want push error without a remote")
	}
	if errors.Is(err, promptErr) {
		t.Fatalf("Publish() error = %v, prompt should have been skipped", err)
	}
	if !strings.Contains(err.Error(), "push -u origin") {
		t.Fatalf("Publish() error = %v, want git push failure", err)
	}
}

func TestFinishInteractiveNoPRUsesHelpfulError(t *testing.T) {
	repoDir := initHotfixRepo(t)
	installNoPRGH(t)

	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &bytes.Buffer{}
	u.Interactive = true

	svc := &Service{
		Git:            git.New(r, repoDir),
		GH:             gh.New(r),
		UI:             u,
		Config:         config.Defaults(),
		RunTitlePrompt: ui.RunTitlePrompt,
		RunProgress:    ui.RunProgress,
	}

	err := svc.Finish(FinishOpts{})
	if err == nil {
		t.Fatal("Finish() error = nil, want helpful no-PR error")
	}

	want := "no PR found for this branch. Run 'git sf hotfix publish' first"
	if err.Error() != want {
		t.Fatalf("Finish() error = %q, want %q", err.Error(), want)
	}
}

func TestFinishClassicBlocksPendingOrCancelledChecks(t *testing.T) {
	tests := []struct {
		name       string
		checksJSON string
		want       string
	}{
		{
			name:       "pending checks block merge",
			checksJSON: `[{"name":"build","state":"PENDING","bucket":"pending"}]`,
			want:       "PR checks still running: build (use --force to override)",
		},
		{
			name:       "cancelled checks block merge",
			checksJSON: `[{"name":"lint","state":"CANCELLED","bucket":"cancel"}]`,
			want:       "PR checks failed: lint (use --force to override)",
		},
		{
			name:       "timed out checks block merge",
			checksJSON: `[{"name":"integration","state":"TIMED_OUT","bucket":"fail"}]`,
			want:       "PR checks failed: integration (use --force to override)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoDir := initHotfixRepo(t)
			installChecksGH(t, tt.checksJSON)

			r := runner.NewRunner(false, false)
			u := ui.New()
			u.Out = &bytes.Buffer{}
			u.Interactive = false

			svc := &Service{
				Git:            git.New(r, repoDir),
				GH:             gh.New(r),
				UI:             u,
				Config:         config.Defaults(),
				RunTitlePrompt: ui.RunTitlePrompt,
				RunProgress:    ui.RunProgress,
			}

			err := svc.Finish(FinishOpts{})
			if err == nil {
				t.Fatal("Finish() error = nil, want checks gate failure")
			}
			if err.Error() != tt.want {
				t.Fatalf("Finish() error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}

func TestDiscardClassicWarnsWhenGHUnavailable(t *testing.T) {
	repoDir := initHotfixRepo(t)
	installGitOnlyPath(t)

	var out bytes.Buffer
	r := runner.NewRunner(false, false)
	u := ui.New()
	u.Out = &out
	u.Interactive = false
	u.AutoConfirm = true

	svc := &Service{
		Git:            git.New(r, repoDir),
		GH:             gh.New(r),
		UI:             u,
		Config:         config.Defaults(),
		RunTitlePrompt: ui.RunTitlePrompt,
		RunProgress:    ui.RunProgress,
	}

	if err := svc.Discard(""); err != nil {
		t.Fatalf("Discard() error = %v", err)
	}

	if !strings.Contains(out.String(), "gh CLI not available — skipping PR close") {
		t.Fatalf("Discard() output = %q, want gh unavailable warning", out.String())
	}
}

func initHotfixRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runGit(t, repoDir, "init", "-b", "main")
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")

	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "init")
	runGit(t, repoDir, "checkout", "-b", "hotfix/test")

	return repoDir
}

func installFakeGH(t *testing.T) {
	t.Helper()

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	script := "#!/bin/sh\nif [ \"$1\" = \"auth\" ] && [ \"$2\" = \"status\" ]; then\n  exit 0\nfi\necho \"unexpected gh command: $*\" >&2\nexit 1\n"
	if err := os.WriteFile(ghPath, []byte(script), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
}

func installGitOnlyPath(t *testing.T) {
	t.Helper()

	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Fatalf("LookPath(git) error = %v", err)
	}

	binDir := t.TempDir()
	gitWrapper := filepath.Join(binDir, "git")
	script := "#!/bin/sh\nexec \"" + gitPath + "\" \"$@\"\n"
	if err := os.WriteFile(gitWrapper, []byte(script), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("PATH", binDir)
}

func installNoPRGH(t *testing.T) {
	t.Helper()

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	script := "#!/bin/sh\nif [ \"$1\" = \"auth\" ] && [ \"$2\" = \"status\" ]; then\n  exit 0\nfi\nif [ \"$1\" = \"pr\" ] && [ \"$2\" = \"view\" ]; then\n  echo \"no pull requests found\" >&2\n  exit 1\nfi\necho \"unexpected gh command: $*\" >&2\nexit 1\n"
	if err := os.WriteFile(ghPath, []byte(script), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
}

func installChecksGH(t *testing.T, checksJSON string) {
	t.Helper()

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"auth\" ] && [ \"$2\" = \"status\" ]; then\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"pr\" ] && [ \"$2\" = \"view\" ]; then\n" +
		"  echo '{\"number\":123,\"title\":\"Hotfix PR\",\"state\":\"OPEN\",\"url\":\"https://example.com/pr/123\",\"isDraft\":false}'\n" +
		"  exit 0\n" +
		"fi\n" +
		"if [ \"$1\" = \"pr\" ] && [ \"$2\" = \"checks\" ]; then\n" +
		"  case \"$*\" in\n" +
		"    *--required*) ;;\n" +
		"    *) echo \"missing --required flag in: $*\" >&2; exit 1 ;;\n" +
		"  esac\n" +
		"  echo '" + checksJSON + "'\n" +
		"  exit 0\n" +
		"fi\n" +
		"echo \"unexpected gh command: $*\" >&2\n" +
		"exit 1\n"
	if err := os.WriteFile(ghPath, []byte(script), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}

	return strings.TrimSpace(string(out))
}
