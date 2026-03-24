package config

import (
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v3"
)

func boolPtr(b bool) *bool { return &b }

func TestDefaults(t *testing.T) {
	cfg := Defaults()
	if cfg.MainBranch != "main" {
		t.Errorf("MainBranch = %q, want %q", cfg.MainBranch, "main")
	}
	if cfg.TagPrefix != "v" {
		t.Errorf("TagPrefix = %q, want %q", cfg.TagPrefix, "v")
	}
	if cfg.MergeStrategy != "squash" {
		t.Errorf("MergeStrategy = %q, want %q", cfg.MergeStrategy, "squash")
	}
	if cfg.DefaultReleaseBump != "minor" {
		t.Errorf("DefaultReleaseBump = %q, want %q", cfg.DefaultReleaseBump, "minor")
	}
	if cfg.FeaturePrefix != "feature/" {
		t.Errorf("FeaturePrefix = %q, want %q", cfg.FeaturePrefix, "feature/")
	}
	if cfg.HotfixPrefix != "hotfix/" {
		t.Errorf("HotfixPrefix = %q, want %q", cfg.HotfixPrefix, "hotfix/")
	}
	if cfg.DraftPROnStart {
		t.Error("DraftPROnStart should default to false")
	}
	if cfg.HotfixAutoRelease {
		t.Error("HotfixAutoRelease should default to false")
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".sfconfig.yml")
	content := []byte("main_branch: develop\nmerge_strategy: rebase\n")
	if err := os.WriteFile(cfgPath, content, 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFromFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MainBranch != "develop" {
		t.Errorf("MainBranch = %q, want %q", cfg.MainBranch, "develop")
	}
	if cfg.MergeStrategy != "rebase" {
		t.Errorf("MergeStrategy = %q, want %q", cfg.MergeStrategy, "rebase")
	}
}

func TestLoadFromFileMissing(t *testing.T) {
	cfg, err := LoadFromFile("/nonexistent/.sfconfig.yml")
	if err != nil {
		t.Fatal("missing file should not error, just return empty config")
	}
	if cfg != nil {
		t.Error("missing file should return nil config")
	}
}

func TestMerge(t *testing.T) {
	base := Defaults()
	global := &PartialConfig{MergeStrategy: "rebase"}
	repo := &PartialConfig{MainBranch: "develop", TagPrefix: "release-"}
	result := Merge(base, global, repo)
	if result.MainBranch != "develop" {
		t.Errorf("MainBranch = %q, want %q (from repo)", result.MainBranch, "develop")
	}
	if result.MergeStrategy != "rebase" {
		t.Errorf("MergeStrategy = %q, want %q (from global)", result.MergeStrategy, "rebase")
	}
	if result.TagPrefix != "release-" {
		t.Errorf("TagPrefix = %q, want %q (from repo)", result.TagPrefix, "release-")
	}
	if result.FeaturePrefix != "feature/" {
		t.Errorf("FeaturePrefix = %q, want %q (from default)", result.FeaturePrefix, "feature/")
	}
}

func TestMergeRepoOverridesGlobal(t *testing.T) {
	base := Defaults()
	global := &PartialConfig{MergeStrategy: "rebase"}
	repo := &PartialConfig{MergeStrategy: "merge"}
	result := Merge(base, global, repo)
	if result.MergeStrategy != "merge" {
		t.Errorf("MergeStrategy = %q, want %q (repo should override global)", result.MergeStrategy, "merge")
	}
}

func TestMergeBoolOverride(t *testing.T) {
	base := Defaults()
	global := &PartialConfig{DraftPROnStart: boolPtr(true)}
	repo := &PartialConfig{HotfixAutoRelease: boolPtr(true)}
	result := Merge(base, global, repo)
	if !result.DraftPROnStart {
		t.Error("DraftPROnStart should be true (from global)")
	}
	if !result.HotfixAutoRelease {
		t.Error("HotfixAutoRelease should be true (from repo)")
	}
}

func TestMergeNilLayers(t *testing.T) {
	base := Defaults()
	result := Merge(base, nil, nil)
	if result != base {
		t.Error("merge with nil layers should return defaults")
	}
}

func TestWriteDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".sfconfig.yml")
	err := WriteDefaults(path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.MainBranch != "main" {
		t.Errorf("MainBranch = %q, want %q", cfg.MainBranch, "main")
	}
}

func TestWriteDefaultsExistingFileErrors(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".sfconfig.yml")
	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	err := WriteDefaults(path)
	if err == nil {
		t.Error("expected error when file already exists")
	}
}
