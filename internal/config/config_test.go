package config

import (
	"os"
	"path/filepath"
	"strings"
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

func TestWriteConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	cfg := Config{
		MainBranch:         "develop",
		TagPrefix:          "release-",
		FeaturePrefix:      "feat/",
		HotfixPrefix:       "fix/",
		MergeStrategy:      "rebase",
		DefaultReleaseBump: "patch",
		DraftPROnStart:     true,
		HotfixAutoRelease:  true,
	}

	if err := WriteConfig(path, cfg); err != nil {
		t.Fatalf("WriteConfig() error = %v", err)
	}

	loaded, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadFromFile() returned nil")
	}

	if loaded.MainBranch != "develop" {
		t.Errorf("MainBranch = %q, want %q", loaded.MainBranch, "develop")
	}
	if loaded.FeaturePrefix != "feat/" {
		t.Errorf("FeaturePrefix = %q, want %q", loaded.FeaturePrefix, "feat/")
	}
	if loaded.MergeStrategy != "rebase" {
		t.Errorf("MergeStrategy = %q, want %q", loaded.MergeStrategy, "rebase")
	}
	if loaded.DraftPROnStart == nil || !*loaded.DraftPROnStart {
		t.Error("DraftPROnStart should be true")
	}
	if loaded.HotfixAutoRelease == nil || !*loaded.HotfixAutoRelease {
		t.Error("HotfixAutoRelease should be true")
	}
}

func TestWritePartialConfigOnlyWritesSetFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	draftPR := true
	partial := PartialConfig{
		MainBranch:     "develop",
		FeaturePrefix:  "feat/",
		HotfixPrefix:   "fix/",
		TagPrefix:      "rel-",
		DraftPROnStart: &draftPR,
	}

	if err := WritePartialConfig(path, partial); err != nil {
		t.Fatalf("WritePartialConfig() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "main_branch") {
		t.Error("expected main_branch in output")
	}
	if !strings.Contains(content, "feature_prefix") {
		t.Error("expected feature_prefix in output")
	}
	if strings.Contains(content, "merge_strategy") {
		t.Error("merge_strategy should not be written (not set in partial)")
	}
	if strings.Contains(content, "default_release_bump") {
		t.Error("default_release_bump should not be written (not set in partial)")
	}
	if strings.Contains(content, "hotfix_auto_release") {
		t.Error("hotfix_auto_release should not be written (not set in partial)")
	}
}

func TestUpdatePartialConfigFilePreservesUnknownFieldsAndComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")

	initial := strings.Join([]string{
		"# repo config",
		"main_branch: main",
		"# custom extension",
		"custom_field: keep-me",
		"merge_strategy: squash",
		"",
	}, "\n")
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}

	if err := UpdatePartialConfigFile(path, PartialConfig{MainBranch: "develop"}); err != nil {
		t.Fatalf("UpdatePartialConfigFile() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "# repo config") {
		t.Errorf("expected top-level comment to be preserved, got %q", content)
	}
	if !strings.Contains(content, "# custom extension") {
		t.Errorf("expected unknown-field comment to be preserved, got %q", content)
	}
	if !strings.Contains(content, "custom_field: keep-me") {
		t.Errorf("expected unknown field to be preserved, got %q", content)
	}
	if !strings.Contains(content, "main_branch: develop") {
		t.Errorf("expected known field to be updated, got %q", content)
	}
	if strings.Contains(content, "merge_strategy:") {
		t.Errorf("expected cleared known field to be removed, got %q", content)
	}
}

func TestUpdatePartialConfigFileCreatesMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	draftPR := false

	if err := UpdatePartialConfigFile(path, PartialConfig{
		MainBranch:     "develop",
		DraftPROnStart: &draftPR,
	}); err != nil {
		t.Fatalf("UpdatePartialConfigFile() error = %v", err)
	}

	loaded, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	if loaded == nil {
		t.Fatal("LoadFromFile() returned nil")
	}
	if loaded.MainBranch != "develop" {
		t.Errorf("MainBranch = %q, want %q", loaded.MainBranch, "develop")
	}
	if loaded.DraftPROnStart == nil || *loaded.DraftPROnStart {
		t.Fatalf("DraftPROnStart = %v, want explicit false pointer", loaded.DraftPROnStart)
	}
}

func TestValidateDefaults(t *testing.T) {
	cfg := Defaults()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() on defaults should succeed, got %v", err)
	}
}

func TestValidateEmptyMainBranch(t *testing.T) {
	cfg := Defaults()
	cfg.MainBranch = ""
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should fail with empty MainBranch")
	}
	if !strings.Contains(err.Error(), "main_branch must not be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateInvalidMergeStrategy(t *testing.T) {
	cfg := Defaults()
	cfg.MergeStrategy = "fast-forward"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should fail with invalid MergeStrategy")
	}
	if !strings.Contains(err.Error(), "invalid merge_strategy") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateValidMergeStrategies(t *testing.T) {
	for _, strategy := range []string{"squash", "merge", "rebase"} {
		cfg := Defaults()
		cfg.MergeStrategy = strategy
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() with MergeStrategy=%q should succeed, got %v", strategy, err)
		}
	}
}

func TestValidateInvalidDefaultReleaseBump(t *testing.T) {
	cfg := Defaults()
	cfg.DefaultReleaseBump = "tiny"
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() should fail with invalid DefaultReleaseBump")
	}
	if !strings.Contains(err.Error(), "invalid default_release_bump") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateValidDefaultReleaseBumps(t *testing.T) {
	for _, bump := range []string{"minor", "patch", "major"} {
		cfg := Defaults()
		cfg.DefaultReleaseBump = bump
		if err := cfg.Validate(); err != nil {
			t.Errorf("Validate() with DefaultReleaseBump=%q should succeed, got %v", bump, err)
		}
	}
}

func TestValidateEmptyPrefixes(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*Config)
		want  string
	}{
		{"empty FeaturePrefix", func(c *Config) { c.FeaturePrefix = "" }, "feature_prefix must not be empty"},
		{"empty HotfixPrefix", func(c *Config) { c.HotfixPrefix = "" }, "hotfix_prefix must not be empty"},
		{"empty TagPrefix", func(c *Config) { c.TagPrefix = "" }, "tag_prefix must not be empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			tt.setup(&cfg)
			err := cfg.Validate()
			if err == nil {
				t.Fatalf("Validate() should fail with %s", tt.name)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() error = %q, want containing %q", err, tt.want)
			}
		})
	}
}
