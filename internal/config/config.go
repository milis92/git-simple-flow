// Package config implements 3-layer configuration loading for git-sf.
// Configuration is resolved by merging layers in order: built-in defaults,
// global config (~/.config/git-sf/config.yml), and repo config (.sfconfig.yml).
// Each layer overrides non-zero values from the layer below.
package config

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// Config holds the fully resolved configuration with all fields populated.
type Config struct {
	MainBranch         string `yaml:"main_branch"`
	TagPrefix          string `yaml:"tag_prefix"`
	FeaturePrefix      string `yaml:"feature_prefix"`
	HotfixPrefix       string `yaml:"hotfix_prefix"`
	MergeStrategy      string `yaml:"merge_strategy"`
	DefaultReleaseBump string `yaml:"default_release_bump"`
	DraftPROnStart     bool   `yaml:"draft_pr_on_start"`
	HotfixAutoRelease  bool   `yaml:"hotfix_auto_release"`
}

// PartialConfig represents a sparse configuration from a single layer (global
// or repo). String fields use zero values to indicate "not set". Bool fields
// use pointers so that nil (not set) is distinguishable from false.
type PartialConfig struct {
	MainBranch         string `yaml:"main_branch"`
	TagPrefix          string `yaml:"tag_prefix"`
	FeaturePrefix      string `yaml:"feature_prefix"`
	HotfixPrefix       string `yaml:"hotfix_prefix"`
	MergeStrategy      string `yaml:"merge_strategy"`
	DefaultReleaseBump string `yaml:"default_release_bump"`
	DraftPROnStart     *bool  `yaml:"draft_pr_on_start"`
	HotfixAutoRelease  *bool  `yaml:"hotfix_auto_release"`
}

// Defaults returns the built-in default configuration.
func Defaults() Config {
	return Config{
		MainBranch:         "main",
		TagPrefix:          "v",
		FeaturePrefix:      "feature/",
		HotfixPrefix:       "hotfix/",
		MergeStrategy:      "squash",
		DefaultReleaseBump: "minor",
		DraftPROnStart:     false,
		HotfixAutoRelease:  false,
	}
}

// LoadFromFile reads and parses a YAML config file at the given path.
// If the file does not exist, it returns (nil, nil) rather than an error.
func LoadFromFile(path string) (*PartialConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var cfg PartialConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Merge applies partial config layers onto a base in order, overriding only
// non-zero string fields and non-nil bool pointer fields. Nil layers are skipped.
func Merge(base Config, layers ...*PartialConfig) Config {
	result := base
	for _, layer := range layers {
		if layer == nil {
			continue
		}
		if layer.MainBranch != "" {
			result.MainBranch = layer.MainBranch
		}
		if layer.TagPrefix != "" {
			result.TagPrefix = layer.TagPrefix
		}
		if layer.FeaturePrefix != "" {
			result.FeaturePrefix = layer.FeaturePrefix
		}
		if layer.HotfixPrefix != "" {
			result.HotfixPrefix = layer.HotfixPrefix
		}
		if layer.MergeStrategy != "" {
			result.MergeStrategy = layer.MergeStrategy
		}
		if layer.DefaultReleaseBump != "" {
			result.DefaultReleaseBump = layer.DefaultReleaseBump
		}
		if layer.DraftPROnStart != nil {
			result.DraftPROnStart = *layer.DraftPROnStart
		}
		if layer.HotfixAutoRelease != nil {
			result.HotfixAutoRelease = *layer.HotfixAutoRelease
		}
	}
	return result
}

// WriteDefaults writes the default configuration as YAML to the given path.
// It returns an error if the file already exists.
func WriteDefaults(path string) error {
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("config file already exists: %s (use --force to overwrite)", path)
	}
	data, err := yaml.Marshal(Defaults())
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// ForceWriteDefaults writes the default configuration as YAML to the given path,
// overwriting any existing file.
func ForceWriteDefaults(path string) error {
	data, err := yaml.Marshal(Defaults())
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
