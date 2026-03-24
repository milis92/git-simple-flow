package gh

import (
	"testing"
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
