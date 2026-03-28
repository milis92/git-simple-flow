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
		bucket string
		want   bool
	}{
		{"pending", true},
		{"pass", false},
		{"fail", false},
		{"skipping", false},
		{"cancel", false},
	}
	for _, tt := range tests {
		t.Run(tt.bucket, func(t *testing.T) {
			got := CheckIsPending(CheckStatus{Bucket: tt.bucket})
			if got != tt.want {
				t.Errorf("CheckIsPending(bucket=%q) = %v, want %v", tt.bucket, got, tt.want)
			}
		})
	}
}

func TestCheckAllowsMerge(t *testing.T) {
	tests := []struct {
		name   string
		bucket string
		want   bool
	}{
		{"pass", "pass", true},
		{"skipping", "skipping", true},
		{"fail", "fail", false},
		{"cancel", "cancel", false},
		{"pending", "pending", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckAllowsMerge(CheckStatus{Bucket: tt.bucket})
			if got != tt.want {
				t.Errorf("CheckAllowsMerge(bucket=%q) = %v, want %v",
					tt.bucket, got, tt.want)
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
				{Name: "build", State: "SUCCESS", Bucket: "pass"},
				{Name: "lint", State: "SKIPPING", Bucket: "skipping"},
			},
		},
		{
			name: "one failing",
			checks: []CheckStatus{
				{Name: "build", State: "SUCCESS", Bucket: "pass"},
				{Name: "test", State: "FAILURE", Bucket: "fail"},
			},
			wantFailing: []string{"test"},
		},
		{
			name: "one pending",
			checks: []CheckStatus{
				{Name: "build", State: "SUCCESS", Bucket: "pass"},
				{Name: "deploy", State: "PENDING", Bucket: "pending"},
			},
			wantPending: []string{"deploy"},
		},
		{
			name: "mixed",
			checks: []CheckStatus{
				{Name: "build", State: "FAILURE", Bucket: "fail"},
				{Name: "lint", State: "PENDING", Bucket: "pending"},
				{Name: "test", State: "SUCCESS", Bucket: "pass"},
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

func TestGetPRChecksPassesRequiredFlag(t *testing.T) {
	installFakeGH(t, `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "checks" ]; then
  case "$*" in
    *--required*) ;;
    *) echo "missing --required flag in: $*" >&2; exit 1 ;;
  esac
  echo '[{"name":"ci","state":"SUCCESS","bucket":"pass"}]'
  exit 0
fi
echo "unexpected gh command: $*" >&2
exit 1
`)

	client := New(runner.NewRunner(false, false))
	checks, err := client.GetPRChecks()
	if err != nil {
		t.Fatalf("GetPRChecks() error = %v", err)
	}
	if len(checks) != 1 || checks[0].Bucket != "pass" {
		t.Fatalf("GetPRChecks() = %v, want one passing check", checks)
	}
}

func TestClosePRPassesBranchSelector(t *testing.T) {
	installFakeGH(t, `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "close" ]; then
  if [ "$3" != "feature/test" ]; then
    echo "expected branch selector 'feature/test', got '$3'" >&2
    exit 1
  fi
  exit 0
fi
echo "unexpected gh command: $*" >&2
exit 1
`)

	client := New(runner.NewRunner(false, false))
	if err := client.ClosePR("feature/test", ""); err != nil {
		t.Fatalf("ClosePR() error = %v", err)
	}
}

func TestClosePRPassesBranchSelectorWithComment(t *testing.T) {
	installFakeGH(t, `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "close" ]; then
  if [ "$3" != "hotfix/urgent" ]; then
    echo "expected branch selector 'hotfix/urgent', got '$3'" >&2
    exit 1
  fi
  if [ "$4" != "--comment" ]; then
    echo "expected --comment flag at \$4, got '$4'" >&2
    exit 1
  fi
  if [ "$5" != "no longer needed" ]; then
    echo "expected comment 'no longer needed', got '$5'" >&2
    exit 1
  fi
  exit 0
fi
echo "unexpected gh command: $*" >&2
exit 1
`)

	client := New(runner.NewRunner(false, false))
	if err := client.ClosePR("hotfix/urgent", "no longer needed"); err != nil {
		t.Fatalf("ClosePR() error = %v", err)
	}
}

func TestMergePRWithMessagePassesFlags(t *testing.T) {
	installFakeGH(t, `#!/bin/sh
if [ "$1" = "pr" ] && [ "$2" = "merge" ]; then
  case "$*" in
    *--merge*) ;;
    *) echo "missing --merge flag in: $*" >&2; exit 1 ;;
  esac
  case "$*" in
    *--subject*) ;;
    *) echo "missing --subject flag in: $*" >&2; exit 1 ;;
  esac
  case "$*" in
    *--body*) ;;
    *) echo "missing --body flag in: $*" >&2; exit 1 ;;
  esac
  exit 0
fi
echo "unexpected gh command: $*" >&2
exit 1
`)

	client := New(runner.NewRunner(false, false))
	if err := client.MergePRWithMessage("merge", "Merge hotfix v1.2.4", ""); err != nil {
		t.Fatalf("MergePRWithMessage() error = %v", err)
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
