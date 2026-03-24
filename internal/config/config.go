package config

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

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

func ForceWriteDefaults(path string) error {
	data, err := yaml.Marshal(Defaults())
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
