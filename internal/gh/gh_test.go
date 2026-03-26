package gh

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/milis92/git-simple-flow/internal/runner"
)

func TestCheckGHInstalled(t *testing.T) {
	err := CheckGHInstalled()
	if err != nil {
		t.Skip("gh not installed, skipping")
	}
}

func TestHumanizeBranchName(t *testing.T) {
	tests := []struct {
		branch string
		prefix string
		want   string
	}{
		{"feature/dark-mode", "feature/", "Dark mode"},
		{"feature/add-auth-system", "feature/", "Add auth system"},
		{"hotfix/fix-crash", "hotfix/", "Fix crash"},
		{"feature/JIRA-123-login", "feature/", "JIRA 123 login"},
	}
	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			got := HumanizeBranchName(tt.branch, tt.prefix)
			if got != tt.want {
				t.Errorf("HumanizeBranchName(%q, %q) = %q, want %q", tt.branch, tt.prefix, got, tt.want)
			}
		})
	}
}

func TestGetCurrentPRWrapsNoPRAsErrNoPR(t *testing.T) {
	installFakeGH(t, `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "view" ]; then
  echo "no pull requests found for branch" >&2
  exit 1
fi
echo "unexpected gh command: $*" >&2
exit 1
`)

	client := New(runner.NewRunner(false, false))
	_, err := client.GetCurrentPR()
	if err == nil {
		t.Fatal("GetCurrentPR() error = nil, want ErrNoPR")
	}
	if !errors.Is(err, ErrNoPR) {
		t.Fatalf("GetCurrentPR() error = %v, want wrapped ErrNoPR", err)
	}
}

func TestGetCurrentPRDoesNotWrapUnexpectedErrorsAsErrNoPR(t *testing.T) {
	installFakeGH(t, `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "view" ]; then
  echo "GraphQL API unavailable" >&2
  exit 1
fi
echo "unexpected gh command: $*" >&2
exit 1
`)

	client := New(runner.NewRunner(false, false))
	_, err := client.GetCurrentPR()
	if err == nil {
		t.Fatal("GetCurrentPR() error = nil, want command error")
	}
	if errors.Is(err, ErrNoPR) {
		t.Fatalf("GetCurrentPR() error = %v, should not wrap ErrNoPR for unrelated failures", err)
	}
}

func installFakeGH(t *testing.T, script string) {
	t.Helper()

	binDir := t.TempDir()
	ghPath := filepath.Join(binDir, "gh")
	if err := os.WriteFile(ghPath, []byte(script), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
}
