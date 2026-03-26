package cmd

import (
	"testing"

	"github.com/milis92/git-simple-flow/internal/config"
	"github.com/milis92/git-simple-flow/internal/ui"
)

func boolPtr(v bool) *bool {
	return &v
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
