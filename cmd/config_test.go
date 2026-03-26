package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/ui"
)

func boolPtr(v bool) *bool {
	return &v
}

func TestLoadPartialConfigSanitizesInvalidEnums(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	content := []byte("main_branch: develop\nmerge_strategy: fast-forward\ndefault_release_bump: tiny\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg, warnings, err := loadPartialConfig(path)
	if err != nil {
		t.Fatalf("loadPartialConfig() error = %v", err)
	}
	if len(warnings) != 2 {
		t.Fatalf("warnings count = %d, want 2", len(warnings))
	}
	if cfg == nil {
		t.Fatal("loadPartialConfig() returned nil config")
	}
	if cfg.MainBranch != "develop" {
		t.Errorf("MainBranch = %q, want %q", cfg.MainBranch, "develop")
	}
	if cfg.MergeStrategy != "" {
		t.Errorf("MergeStrategy = %q, want empty after sanitization", cfg.MergeStrategy)
	}
	if cfg.DefaultReleaseBump != "" {
		t.Errorf("DefaultReleaseBump = %q, want empty after sanitization", cfg.DefaultReleaseBump)
	}
}

func TestShouldUseInitWizard(t *testing.T) {
	tests := []struct {
		name string
		ui   *ui.UI
		want bool
	}{
		{
			name: "interactive prompt-enabled mode uses wizard",
			ui: &ui.UI{
				Interactive: true,
			},
			want: true,
		},
		{
			name: "auto confirm skips wizard",
			ui: &ui.UI{
				Interactive: true,
				AutoConfirm: true,
			},
			want: false,
		},
		{
			name: "non interactive skips wizard",
			ui: &ui.UI{
				Interactive: false,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldUseInitWizard(tt.ui); got != tt.want {
				t.Fatalf("shouldUseInitWizard() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitWizardDefaultsSeedsFromExistingConfigWhenForce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".sfconfig.yml")
	content := []byte("main_branch: develop\nmerge_strategy: rebase\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got := initWizardDefaults(path, true)

	if got.MainBranch != "develop" {
		t.Errorf("MainBranch = %q, want %q from existing config", got.MainBranch, "develop")
	}
	if got.MergeStrategy != "rebase" {
		t.Errorf("MergeStrategy = %q, want %q from existing config", got.MergeStrategy, "rebase")
	}
	// Fields not in the existing file should fall back to built-in defaults
	defaults := config.Defaults()
	if got.FeaturePrefix != defaults.FeaturePrefix {
		t.Errorf("FeaturePrefix = %q, want default %q", got.FeaturePrefix, defaults.FeaturePrefix)
	}
}

func TestInitWizardDefaultsUsesBarDefaultsWithoutForce(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".sfconfig.yml")
	content := []byte("main_branch: develop\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got := initWizardDefaults(path, false)
	defaults := config.Defaults()

	if got.MainBranch != defaults.MainBranch {
		t.Errorf("MainBranch = %q, want default %q (force=false should ignore existing file)", got.MainBranch, defaults.MainBranch)
	}
}

func TestInitWizardDefaultsFallsBackOnMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nonexistent.yml")

	got := initWizardDefaults(path, true)
	defaults := config.Defaults()

	if got.MainBranch != defaults.MainBranch {
		t.Errorf("MainBranch = %q, want default %q (missing file should fall back)", got.MainBranch, defaults.MainBranch)
	}
}

func TestBuildRepoConfigForEditPreservesUntouchedFields(t *testing.T) {
	inherited := config.Config{
		MainBranch:         "main",
		FeaturePrefix:      "feature/",
		HotfixPrefix:       "hotfix/",
		TagPrefix:          "v",
		MergeStrategy:      "squash",
		DefaultReleaseBump: "minor",
		DraftPROnStart:     false,
		HotfixAutoRelease:  false,
	}
	existing := &config.PartialConfig{
		MainBranch:        "develop",
		MergeStrategy:     "rebase",
		HotfixAutoRelease: boolPtr(true),
	}

	result := ui.InitFormResult{
		MainBranch:         "main",
		FeaturePrefix:      "feat/",
		HotfixPrefix:       "hotfix/",
		TagPrefix:          "v",
		MergeStrategy:      "merge",
		DefaultReleaseBump: "patch",
		DraftPR:            false,
		HotfixAutoRelease:  false,
	}

	updated := buildRepoConfigForEdit(inherited, existing, result)

	if updated.MainBranch != "" {
		t.Errorf("MainBranch = %q, want empty to inherit", updated.MainBranch)
	}
	if updated.FeaturePrefix != "feat/" {
		t.Errorf("FeaturePrefix = %q, want %q", updated.FeaturePrefix, "feat/")
	}
	if updated.HotfixPrefix != "" {
		t.Errorf("HotfixPrefix = %q, want empty to inherit", updated.HotfixPrefix)
	}
	if updated.TagPrefix != "" {
		t.Errorf("TagPrefix = %q, want empty to inherit", updated.TagPrefix)
	}
	if updated.MergeStrategy != "merge" {
		t.Errorf("MergeStrategy = %q, want %q", updated.MergeStrategy, "merge")
	}
	if updated.DefaultReleaseBump != "patch" {
		t.Errorf("DefaultReleaseBump = %q, want %q", updated.DefaultReleaseBump, "patch")
	}
	if updated.DraftPROnStart != nil {
		t.Errorf("DraftPROnStart = %v, want nil to inherit", *updated.DraftPROnStart)
	}
	if updated.HotfixAutoRelease != nil {
		t.Errorf("HotfixAutoRelease = %v, want nil to inherit", *updated.HotfixAutoRelease)
	}
}

func TestBuildRepoConfigForEditDoesNotPromoteInheritedValues(t *testing.T) {
	inherited := config.Config{
		MainBranch:         "develop",
		FeaturePrefix:      "feat/",
		HotfixPrefix:       "fix/",
		TagPrefix:          "rel-",
		MergeStrategy:      "squash",
		DefaultReleaseBump: "minor",
		DraftPROnStart:     true,
		HotfixAutoRelease:  false,
	}

	result := ui.InitFormResult{
		MainBranch:         "develop",
		FeaturePrefix:      "feat/",
		HotfixPrefix:       "fix/",
		TagPrefix:          "rel-",
		MergeStrategy:      "squash",
		DefaultReleaseBump: "minor",
		DraftPR:            true,
		HotfixAutoRelease:  false,
	}

	updated := buildRepoConfigForEdit(inherited, nil, result)

	if updated.MainBranch != "" {
		t.Errorf("MainBranch = %q, want empty", updated.MainBranch)
	}
	if updated.FeaturePrefix != "" {
		t.Errorf("FeaturePrefix = %q, want empty", updated.FeaturePrefix)
	}
	if updated.HotfixPrefix != "" {
		t.Errorf("HotfixPrefix = %q, want empty", updated.HotfixPrefix)
	}
	if updated.TagPrefix != "" {
		t.Errorf("TagPrefix = %q, want empty", updated.TagPrefix)
	}
	if updated.MergeStrategy != "" {
		t.Errorf("MergeStrategy = %q, want empty", updated.MergeStrategy)
	}
	if updated.DefaultReleaseBump != "" {
		t.Errorf("DefaultReleaseBump = %q, want empty", updated.DefaultReleaseBump)
	}
	if updated.DraftPROnStart != nil {
		t.Errorf("DraftPROnStart = %v, want nil", *updated.DraftPROnStart)
	}
	if updated.HotfixAutoRelease != nil {
		t.Errorf("HotfixAutoRelease = %v, want nil", *updated.HotfixAutoRelease)
	}
}

func TestBuildRepoConfigForEditPreservesRepoOverridesWhenStillSelected(t *testing.T) {
	inherited := config.Config{
		MainBranch:         "main",
		FeaturePrefix:      "feature/",
		HotfixPrefix:       "hotfix/",
		TagPrefix:          "v",
		MergeStrategy:      "squash",
		DefaultReleaseBump: "minor",
		DraftPROnStart:     false,
		HotfixAutoRelease:  false,
	}
	existing := &config.PartialConfig{
		MergeStrategy:      "rebase",
		DefaultReleaseBump: "patch",
		HotfixAutoRelease:  boolPtr(true),
	}

	result := ui.InitFormResult{
		MainBranch:         "main",
		FeaturePrefix:      "feature/",
		HotfixPrefix:       "hotfix/",
		TagPrefix:          "v",
		MergeStrategy:      "rebase",
		DefaultReleaseBump: "patch",
		DraftPR:            false,
		HotfixAutoRelease:  true,
	}

	updated := buildRepoConfigForEdit(inherited, existing, result)

	if updated.MergeStrategy != "rebase" {
		t.Errorf("MergeStrategy = %q, want %q", updated.MergeStrategy, "rebase")
	}
	if updated.DefaultReleaseBump != "patch" {
		t.Errorf("DefaultReleaseBump = %q, want %q", updated.DefaultReleaseBump, "patch")
	}
	if updated.HotfixAutoRelease == nil || !*updated.HotfixAutoRelease {
		t.Error("HotfixAutoRelease should preserve the repo override")
	}
}

func TestBuildRepoConfigForEditPreservesExplicitRepoPinsEqualToInherited(t *testing.T) {
	inherited := config.Config{
		MainBranch:         "main",
		FeaturePrefix:      "feature/",
		HotfixPrefix:       "hotfix/",
		TagPrefix:          "v",
		MergeStrategy:      "squash",
		DefaultReleaseBump: "minor",
		DraftPROnStart:     false,
		HotfixAutoRelease:  false,
	}
	existing := &config.PartialConfig{
		MergeStrategy:  "squash",
		DraftPROnStart: boolPtr(false),
	}

	result := ui.InitFormResult{
		MainBranch:         "main",
		FeaturePrefix:      "feature/",
		HotfixPrefix:       "hotfix/",
		TagPrefix:          "v",
		MergeStrategy:      "squash",
		DefaultReleaseBump: "minor",
		DraftPR:            false,
		HotfixAutoRelease:  false,
	}

	updated := buildRepoConfigForEdit(inherited, existing, result)

	if updated.MergeStrategy != "squash" {
		t.Errorf("MergeStrategy = %q, want %q to preserve explicit repo pin", updated.MergeStrategy, "squash")
	}
	if updated.DraftPROnStart == nil || *updated.DraftPROnStart {
		t.Fatalf("DraftPROnStart = %v, want explicit false pointer to preserve repo pin", updated.DraftPROnStart)
	}
}

func TestBuildRepoConfigForEditClearsOverrideWhenChangedBackToInherited(t *testing.T) {
	inherited := config.Config{
		MainBranch:         "main",
		FeaturePrefix:      "feature/",
		HotfixPrefix:       "hotfix/",
		TagPrefix:          "v",
		MergeStrategy:      "squash",
		DefaultReleaseBump: "minor",
		DraftPROnStart:     false,
		HotfixAutoRelease:  false,
	}
	existing := &config.PartialConfig{
		MergeStrategy:  "rebase",
		DraftPROnStart: boolPtr(true),
	}

	result := ui.InitFormResult{
		MainBranch:         "main",
		FeaturePrefix:      "feature/",
		HotfixPrefix:       "hotfix/",
		TagPrefix:          "v",
		MergeStrategy:      "squash",
		DefaultReleaseBump: "minor",
		DraftPR:            false,
		HotfixAutoRelease:  false,
	}

	updated := buildRepoConfigForEdit(inherited, existing, result)

	if updated.MergeStrategy != "" {
		t.Errorf("MergeStrategy = %q, want empty to clear repo override", updated.MergeStrategy)
	}
	if updated.DraftPROnStart != nil {
		t.Fatalf("DraftPROnStart = %v, want nil to clear repo override", *updated.DraftPROnStart)
	}
}

func TestConfigBoolSource(t *testing.T) {
	repo := &config.PartialConfig{DraftPROnStart: boolPtr(false)}
	global := &config.PartialConfig{DraftPROnStart: boolPtr(true)}

	if got := configBoolSource(global, repo, func(c *config.PartialConfig) *bool { return c.DraftPROnStart }); got != repoConfigSource {
		t.Errorf("configBoolSource() = %q, want %q", got, repoConfigSource)
	}

	if got := configBoolSource(global, nil, func(c *config.PartialConfig) *bool { return c.DraftPROnStart }); got != globalConfigSource {
		t.Errorf("configBoolSource() = %q, want %q", got, globalConfigSource)
	}

	if got := configBoolSource(nil, nil, func(c *config.PartialConfig) *bool { return c.DraftPROnStart }); got != defaultConfigSource {
		t.Errorf("configBoolSource() = %q, want %q", got, defaultConfigSource)
	}
}
