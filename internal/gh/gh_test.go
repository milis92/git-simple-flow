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

func TestCheckIsPending(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{"in_progress", true},
		{"queued", true},
		{"completed", false},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := CheckIsPending(CheckStatus{Status: tt.status})
			if got != tt.want {
				t.Errorf("CheckIsPending(status=%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestCheckAllowsMerge(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		conclusion string
		want       bool
	}{
		{"success", "completed", "success", true},
		{"neutral", "completed", "neutral", true},
		{"skipped", "completed", "skipped", true},
		{"failure", "completed", "failure", false},
		{"cancelled", "completed", "cancelled", false},
		{"timed_out", "completed", "timed_out", false},
		{"action_required", "completed", "action_required", false},
		{"stale", "completed", "stale", false},
		{"pending blocks", "in_progress", "success", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckAllowsMerge(CheckStatus{Status: tt.status, Conclusion: tt.conclusion})
			if got != tt.want {
				t.Errorf("CheckAllowsMerge(status=%q, conclusion=%q) = %v, want %v",
					tt.status, tt.conclusion, got, tt.want)
			}
		})
	}
}

func TestClassifyChecks(t *testing.T) {
	tests := []struct {
		name        string
		checks      []CheckStatus
		wantFailing []string
		wantPending []string
	}{
		{
			name:   "empty",
			checks: nil,
		},
		{
			name: "all passing",
			checks: []CheckStatus{
				{Name: "build", Status: "completed", Conclusion: "success"},
				{Name: "lint", Status: "completed", Conclusion: "neutral"},
			},
		},
		{
			name: "one failing",
			checks: []CheckStatus{
				{Name: "build", Status: "completed", Conclusion: "success"},
				{Name: "test", Status: "completed", Conclusion: "failure"},
			},
			wantFailing: []string{"test"},
		},
		{
			name: "one pending",
			checks: []CheckStatus{
				{Name: "build", Status: "completed", Conclusion: "success"},
				{Name: "deploy", Status: "in_progress"},
			},
			wantPending: []string{"deploy"},
		},
		{
			name: "mixed",
			checks: []CheckStatus{
				{Name: "build", Status: "completed", Conclusion: "failure"},
				{Name: "lint", Status: "in_progress"},
				{Name: "test", Status: "completed", Conclusion: "success"},
			},
			wantFailing: []string{"build"},
			wantPending: []string{"lint"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			failing, pending := ClassifyChecks(tt.checks)
			if !slicesEqual(failing, tt.wantFailing) {
				t.Errorf("ClassifyChecks() failing = %v, want %v", failing, tt.wantFailing)
			}
			if !slicesEqual(pending, tt.wantPending) {
				t.Errorf("ClassifyChecks() pending = %v, want %v", pending, tt.wantPending)
			}
		})
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
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
