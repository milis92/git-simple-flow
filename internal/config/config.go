// Package config implements 3-layer configuration loading for git-sf.
// Configuration is resolved by merging layers in order: built-in defaults,
// global config (~/.config/git-sf/config.yml), and repo config (.sfconfig.yml).
// Each layer overrides non-zero values from the layer below.
package config

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Config holds the fully resolved configuration with all fields populated.
type Config struct {
	MainBranch            string `yaml:"main_branch"`
	TagPrefix             string `yaml:"tag_prefix"`
	FeaturePrefix         string `yaml:"feature_prefix"`
	HotfixPrefix          string `yaml:"hotfix_prefix"`
	MergeStrategy         string `yaml:"merge_strategy"`
	DefaultReleaseBump    string `yaml:"default_release_bump"`
	DraftPROnStart        bool   `yaml:"draft_pr_on_start"`
	HotfixAutoRelease     bool   `yaml:"hotfix_auto_release"`
	PrereleaseEnabled     bool   `yaml:"prerelease_enabled"`
	DefaultPrereleaseBump string `yaml:"default_prerelease_bump"`
	PrereleaseSuffix      string `yaml:"prerelease_suffix"`
}

// PartialConfig represents a sparse configuration from a single layer (global
// or repo). String fields use zero values to indicate "not set". Bool fields
// use pointers so that nil (not set) is distinguishable from false.
type PartialConfig struct {
	MainBranch            string `yaml:"main_branch,omitempty"`
	TagPrefix             string `yaml:"tag_prefix,omitempty"`
	FeaturePrefix         string `yaml:"feature_prefix,omitempty"`
	HotfixPrefix          string `yaml:"hotfix_prefix,omitempty"`
	MergeStrategy         string `yaml:"merge_strategy,omitempty"`
	DefaultReleaseBump    string `yaml:"default_release_bump,omitempty"`
	DraftPROnStart        *bool  `yaml:"draft_pr_on_start,omitempty"`
	HotfixAutoRelease     *bool  `yaml:"hotfix_auto_release,omitempty"`
	PrereleaseEnabled     *bool  `yaml:"prerelease_enabled,omitempty"`
	DefaultPrereleaseBump string `yaml:"default_prerelease_bump,omitempty"`
	PrereleaseSuffix      string `yaml:"prerelease_suffix,omitempty"`
}

func isValidMergeStrategy(value string) bool {
	switch value {
	case "squash", "merge", "rebase":
		return true
	default:
		return false
	}
}

func isValidReleaseBump(value string) bool {
	switch value {
	case "minor", "patch", "major":
		return true
	default:
		return false
	}
}

func isValidPrereleaseSuffix(value string) bool {
	if value == "" {
		return false
	}
	for _, c := range value {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// Validate checks that enum-like fields contain valid values and that
// required fields are non-empty. It returns an error describing the first
// invalid field it finds.
func (c Config) Validate() error {
	if c.MainBranch == "" {
		return fmt.Errorf("main_branch must not be empty")
	}
	if !isValidMergeStrategy(c.MergeStrategy) {
		return fmt.Errorf("invalid merge_strategy %q: must be squash, merge, or rebase", c.MergeStrategy)
	}
	if !isValidReleaseBump(c.DefaultReleaseBump) {
		return fmt.Errorf("invalid default_release_bump %q: must be minor, patch, or major", c.DefaultReleaseBump)
	}
	if c.FeaturePrefix == "" {
		return fmt.Errorf("feature_prefix must not be empty")
	}
	if c.HotfixPrefix == "" {
		return fmt.Errorf("hotfix_prefix must not be empty")
	}
	if c.TagPrefix == "" {
		return fmt.Errorf("tag_prefix must not be empty")
	}
	if !isValidReleaseBump(c.DefaultPrereleaseBump) {
		return fmt.Errorf("invalid default_prerelease_bump %q: must be minor, patch, or major", c.DefaultPrereleaseBump)
	}
	if !isValidPrereleaseSuffix(c.PrereleaseSuffix) {
		return fmt.Errorf("invalid prerelease_suffix %q: must be non-empty lowercase alphanumeric", c.PrereleaseSuffix)
	}
	return nil
}

// Defaults returns the built-in default configuration.
func Defaults() Config {
	return Config{
		MainBranch:            "main",
		TagPrefix:             "v",
		FeaturePrefix:         "feature/",
		HotfixPrefix:          "hotfix/",
		MergeStrategy:         "squash",
		DefaultReleaseBump:    "minor",
		DraftPROnStart:        false,
		HotfixAutoRelease:     false,
		PrereleaseEnabled:     false,
		DefaultPrereleaseBump: "patch",
		PrereleaseSuffix:      "beta",
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

// SanitizePartial trims whitespace from string fields, clears blank-only
// values, and removes invalid enum-like fields from a partial config.
// Returned warnings describe each ignored field so callers can surface them.
func SanitizePartial(cfg *PartialConfig) (*PartialConfig, []error) {
	if cfg == nil {
		return nil, nil
	}

	sanitized := *cfg
	var warnings []error

	// Trim whitespace and clear blank-only values.
	sanitized.MainBranch = sanitizeStringField(sanitized.MainBranch, "main_branch", &warnings)
	sanitized.TagPrefix = sanitizeStringField(sanitized.TagPrefix, "tag_prefix", &warnings)
	sanitized.FeaturePrefix = sanitizeStringField(sanitized.FeaturePrefix, "feature_prefix", &warnings)
	sanitized.HotfixPrefix = sanitizeStringField(sanitized.HotfixPrefix, "hotfix_prefix", &warnings)
	sanitized.MergeStrategy = sanitizeStringField(sanitized.MergeStrategy, "merge_strategy", &warnings)
	sanitized.DefaultReleaseBump = sanitizeStringField(sanitized.DefaultReleaseBump, "default_release_bump", &warnings)

	if sanitized.MergeStrategy != "" && !isValidMergeStrategy(sanitized.MergeStrategy) {
		warnings = append(warnings, fmt.Errorf(
			"invalid merge_strategy %q: must be squash, merge, or rebase",
			sanitized.MergeStrategy,
		))
		sanitized.MergeStrategy = ""
	}

	if sanitized.DefaultReleaseBump != "" && !isValidReleaseBump(sanitized.DefaultReleaseBump) {
		warnings = append(warnings, fmt.Errorf(
			"invalid default_release_bump %q: must be minor, patch, or major",
			sanitized.DefaultReleaseBump,
		))
		sanitized.DefaultReleaseBump = ""
	}

	sanitized.DefaultPrereleaseBump = sanitizeStringField(sanitized.DefaultPrereleaseBump, "default_prerelease_bump", &warnings)
	sanitized.PrereleaseSuffix = sanitizeStringField(sanitized.PrereleaseSuffix, "prerelease_suffix", &warnings)

	if sanitized.DefaultPrereleaseBump != "" && !isValidReleaseBump(sanitized.DefaultPrereleaseBump) {
		warnings = append(warnings, fmt.Errorf(
			"invalid default_prerelease_bump %q: must be minor, patch, or major",
			sanitized.DefaultPrereleaseBump,
		))
		sanitized.DefaultPrereleaseBump = ""
	}

	if sanitized.PrereleaseSuffix != "" && !isValidPrereleaseSuffix(sanitized.PrereleaseSuffix) {
		warnings = append(warnings, fmt.Errorf(
			"invalid prerelease_suffix %q: must be non-empty lowercase alphanumeric",
			sanitized.PrereleaseSuffix,
		))
		sanitized.PrereleaseSuffix = ""
	}

	return &sanitized, warnings
}

func sanitizeStringField(value, name string, warnings *[]error) string {
	trimmed := strings.TrimSpace(value)
	if value != "" && trimmed == "" {
		*warnings = append(*warnings, fmt.Errorf("blank %s (ignoring)", name))
	}
	return trimmed
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
		if layer.PrereleaseEnabled != nil {
			result.PrereleaseEnabled = *layer.PrereleaseEnabled
		}
		if layer.DefaultPrereleaseBump != "" {
			result.DefaultPrereleaseBump = layer.DefaultPrereleaseBump
		}
		if layer.PrereleaseSuffix != "" {
			result.PrereleaseSuffix = layer.PrereleaseSuffix
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

// WriteConfig writes a full Config as YAML to the given path, creating or
// overwriting the file.
func WriteConfig(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// WritePartialConfig writes only the non-zero fields of a PartialConfig as YAML
// to the given path, creating or overwriting the file.
func WritePartialConfig(path string, cfg PartialConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// UpdatePartialConfigFile updates known PartialConfig keys in-place while
// preserving unknown keys and YAML comments in the existing file.
func UpdatePartialConfigFile(path string, cfg PartialConfig) error {
	doc, err := loadConfigDocument(path)
	if err != nil {
		return err
	}

	root := doc.Content[0]
	setStringField(root, "main_branch", cfg.MainBranch)
	setStringField(root, "tag_prefix", cfg.TagPrefix)
	setStringField(root, "feature_prefix", cfg.FeaturePrefix)
	setStringField(root, "hotfix_prefix", cfg.HotfixPrefix)
	setStringField(root, "merge_strategy", cfg.MergeStrategy)
	setStringField(root, "default_release_bump", cfg.DefaultReleaseBump)
	setBoolField(root, "draft_pr_on_start", cfg.DraftPROnStart)
	setBoolField(root, "hotfix_auto_release", cfg.HotfixAutoRelease)
	setBoolField(root, "prerelease_enabled", cfg.PrereleaseEnabled)
	setStringField(root, "default_prerelease_bump", cfg.DefaultPrereleaseBump)
	setStringField(root, "prerelease_suffix", cfg.PrereleaseSuffix)

	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}

	return os.WriteFile(path, out.Bytes(), 0644)
}

func loadConfigDocument(path string) (*yaml.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newConfigDocument(), nil
		}
		return nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return newConfigDocument(), nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc.Kind == 0 {
		return newConfigDocument(), nil
	}
	if doc.Kind != yaml.DocumentNode {
		return nil, fmt.Errorf("config file must contain a YAML document")
	}
	if len(doc.Content) == 0 {
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("config file must contain a mapping at the top level")
	}

	return &doc, nil
}

func newConfigDocument() *yaml.Node {
	return &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{Kind: yaml.MappingNode, Tag: "!!map"},
		},
	}
}

func setStringField(root *yaml.Node, key, value string) {
	if value == "" {
		removeMapEntry(root, key)
		return
	}
	setScalarField(root, key, value, "!!str")
}

func setBoolField(root *yaml.Node, key string, value *bool) {
	if value == nil {
		removeMapEntry(root, key)
		return
	}
	setScalarField(root, key, strconv.FormatBool(*value), "!!bool")
}

func setScalarField(root *yaml.Node, key, value, tag string) {
	_, node, idx := mapEntry(root, key)
	if idx >= 0 {
		node.Kind = yaml.ScalarNode
		node.Tag = tag
		node.Value = value
		return
	}

	root.Content = append(root.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value},
	)
}

func removeMapEntry(root *yaml.Node, key string) {
	_, _, idx := mapEntry(root, key)
	if idx < 0 {
		return
	}
	root.Content = append(root.Content[:idx], root.Content[idx+2:]...)
}

func mapEntry(root *yaml.Node, key string) (*yaml.Node, *yaml.Node, int) {
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == key {
			return root.Content[i], root.Content[i+1], i
		}
	}
	return nil, nil, -1
}
