package ui

import (
	"testing"

	"github.com/milis92/git-simple-flow/internal/config"
)

func TestInitFormResultToConfig(t *testing.T) {
	result := InitFormResult{
		MainBranch:         "develop",
		FeaturePrefix:      "feat/",
		HotfixPrefix:       "fix/",
		TagPrefix:          "v",
		MergeStrategy:      "rebase",
		DefaultReleaseBump: "patch",
		DraftPR:            true,
		HotfixAutoRelease:  true,
	}

	cfg := result.ToPartialConfig()

	if cfg.MainBranch != "develop" {
		t.Errorf("MainBranch = %q, want %q", cfg.MainBranch, "develop")
	}
	if cfg.FeaturePrefix != "feat/" {
		t.Errorf("FeaturePrefix = %q, want %q", cfg.FeaturePrefix, "feat/")
	}
	if cfg.HotfixPrefix != "fix/" {
		t.Errorf("HotfixPrefix = %q, want %q", cfg.HotfixPrefix, "fix/")
	}
	if cfg.TagPrefix != "v" {
		t.Errorf("TagPrefix = %q, want %q", cfg.TagPrefix, "v")
	}
	if cfg.MergeStrategy != "rebase" {
		t.Errorf("MergeStrategy = %q, want %q", cfg.MergeStrategy, "rebase")
	}
	if cfg.DefaultReleaseBump != "patch" {
		t.Errorf("DefaultReleaseBump = %q, want %q", cfg.DefaultReleaseBump, "patch")
	}
	if cfg.DraftPROnStart == nil || !*cfg.DraftPROnStart {
		t.Error("DraftPROnStart should be true")
	}
	if cfg.HotfixAutoRelease == nil || !*cfg.HotfixAutoRelease {
		t.Error("HotfixAutoRelease should be true")
	}
}

func TestInitFormResultDefaults(t *testing.T) {
	defaults := config.Defaults()
	result := InitFormResultFromDefaults(defaults)

	if result.MainBranch != "main" {
		t.Errorf("MainBranch = %q, want %q", result.MainBranch, "main")
	}
	if result.FeaturePrefix != "feature/" {
		t.Errorf("FeaturePrefix = %q, want %q", result.FeaturePrefix, "feature/")
	}
	if result.MergeStrategy != "squash" {
		t.Errorf("MergeStrategy = %q, want %q", result.MergeStrategy, "squash")
	}
	if result.DefaultReleaseBump != "minor" {
		t.Errorf("DefaultReleaseBump = %q, want %q", result.DefaultReleaseBump, "minor")
	}
	if result.HotfixAutoRelease {
		t.Error("HotfixAutoRelease should default to false")
	}
}

func TestInputPromptResultTitle(t *testing.T) {
	result := InputPromptResult{
		Title: "My PR Title",
		Body:  "Some body text",
	}
	if result.Title != "My PR Title" {
		t.Errorf("Title = %q, want %q", result.Title, "My PR Title")
	}
}
