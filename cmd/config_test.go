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
		MainBranch:     "main",
		FeaturePrefix:  "feature/",
		HotfixPrefix:   "hotfix/",
		TagPrefix:      "v",
		DraftPROnStart: false,
	}
	existing := &config.PartialConfig{
		MainBranch:        "develop",
		MergeStrategy:     "rebase",
		HotfixAutoRelease: boolPtr(true),
	}

	result := ui.InitFormResult{
		MainBranch:    "main",
		FeaturePrefix: "feat/",
		HotfixPrefix:  "hotfix/",
		TagPrefix:     "v",
		DraftPR:       false,
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
	if updated.DraftPROnStart != nil {
		t.Errorf("DraftPROnStart = %v, want nil to inherit", *updated.DraftPROnStart)
	}
	if updated.MergeStrategy != "rebase" {
		t.Errorf("MergeStrategy = %q, want untouched value %q", updated.MergeStrategy, "rebase")
	}
	if updated.HotfixAutoRelease == nil || !*updated.HotfixAutoRelease {
		t.Error("HotfixAutoRelease should be preserved")
	}
}

func TestBuildRepoConfigForEditDoesNotPromoteInheritedValues(t *testing.T) {
	inherited := config.Config{
		MainBranch:     "develop",
		FeaturePrefix:  "feat/",
		HotfixPrefix:   "fix/",
		TagPrefix:      "rel-",
		DraftPROnStart: true,
	}

	result := ui.InitFormResult{
		MainBranch:    "develop",
		FeaturePrefix: "feat/",
		HotfixPrefix:  "fix/",
		TagPrefix:     "rel-",
		DraftPR:       true,
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
	if updated.DraftPROnStart != nil {
		t.Errorf("DraftPROnStart = %v, want nil", *updated.DraftPROnStart)
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
