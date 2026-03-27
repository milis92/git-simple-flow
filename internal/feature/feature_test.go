package feature

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/gh"
	"github.com/milis92/git-simple-flow/internal/git"
	"github.com/milis92/git-simple-flow/internal/runner"
	"github.com/milis92/git-simple-flow/internal/ui"
)

func TestPublishPromptsBeforePush(t *testing.T) {
	repoDir := initFeatureRepo(t)
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
	repoDir := initFeatureRepo(t)
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
	repoDir := initFeatureRepo(t)
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

func TestFinishInteractiveCancelsInFlightMerge(t *testing.T) {
	repoDir := initFeatureRepo(t)
	mergeStarted := filepath.Join(t.TempDir(), "merge-started")
	mergeDone := filepath.Join(t.TempDir(), "merge-done")
	installFinishGH(t, mergeStarted, mergeDone)

	r := runner.NewRunner(false, false)
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
		RunProgress: func(_ string, _ string, _ []ui.StepDef, workflow func(context.Context, ui.StepCallbacks) error) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			errCh := make(chan error, 1)
			go func() {
				errCh <- workflow(ctx, ui.StepCallbacks{
					Start: func() {},
					Done:  func() {},
					Fail:  func(string) {},
				})
			}()

			waitForFile(t, mergeStarted, time.Second)
			cancel()

			select {
			case err := <-errCh:
				return err
			case <-time.After(2 * time.Second):
				t.Fatal("workflow did not stop after cancellation")
				return nil
			}
		},
	}

	err := svc.Finish(FinishOpts{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Finish() error = %v, want context.Canceled", err)
	}

	if _, statErr := os.Stat(mergeDone); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("merge completion marker should not exist after cancellation, stat err = %v", statErr)
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
			repoDir := initFeatureRepo(t)
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

func initFeatureRepo(t *testing.T) string {
	t.Helper()

	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	runGit(t, repoDir, "config", "user.name", "Test User")
	runGit(t, repoDir, "config", "user.email", "test@example.com")

	if err := os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("test\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	runGit(t, repoDir, "add", "README.md")
	runGit(t, repoDir, "commit", "-m", "init")
	runGit(t, repoDir, "checkout", "-b", "feature/test")

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

func installFinishGH(t *testing.T, mergeStartedPath, mergeDonePath string) {
	t.Helper()

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	script := `#!/bin/sh
if [ "$1" = "auth" ] && [ "$2" = "status" ]; then
  exit 0
fi
if [ "$1" = "pr" ] && [ "$2" = "view" ]; then
  echo '{"number":123,"title":"Feature PR","state":"OPEN","url":"https://example.com/pr/123","isDraft":false}'
  exit 0
fi
if [ "$1" = "pr" ] && [ "$2" = "checks" ]; then
  case "$*" in
    *--required*) ;;
    *) echo "missing --required flag in: $*" >&2; exit 1 ;;
  esac
  echo '[{"name":"ci","state":"SUCCESS","bucket":"pass"}]'
  exit 0
fi
if [ "$1" = "pr" ] && [ "$2" = "merge" ]; then
  exec env GO_WANT_FEATURE_HELPER_PROCESS=1 MERGE_STARTED_FILE="$MERGE_STARTED_FILE" MERGE_DONE_FILE="$MERGE_DONE_FILE" "$GIT_SF_TEST_HELPER" -test.run=TestFeatureHelperProcess -- merge-helper
fi
echo "unexpected gh command: $*" >&2
exit 1
`
	if err := os.WriteFile(ghPath, []byte(script), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
	t.Setenv("GIT_SF_TEST_HELPER", os.Args[0])
	t.Setenv("MERGE_STARTED_FILE", mergeStartedPath)
	t.Setenv("MERGE_DONE_FILE", mergeDonePath)
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
		"  echo '{\"number\":123,\"title\":\"Feature PR\",\"state\":\"OPEN\",\"url\":\"https://example.com/pr/123\",\"isDraft\":false}'\n" +
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

func waitForFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}

		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("timed out waiting for %s", path)
}

func TestFeatureHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_FEATURE_HELPER_PROCESS") != "1" {
		return
	}

	args := helperArgs(os.Args)
	if len(args) != 1 || args[0] != "merge-helper" {
		os.Exit(2)
	}

	if err := os.WriteFile(os.Getenv("MERGE_STARTED_FILE"), []byte("started"), 0644); err != nil {
		os.Exit(1)
	}

	time.Sleep(2 * time.Second)
	if err := os.WriteFile(os.Getenv("MERGE_DONE_FILE"), []byte("done"), 0644); err != nil {
		os.Exit(1)
	}

	os.Exit(0)
}

func helperArgs(args []string) []string {
	for i, arg := range args {
		if arg == "--" {
			return args[i+1:]
		}
	}

	return nil
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
