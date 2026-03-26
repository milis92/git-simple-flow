package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/milis92/git-simple-flow/internal/config"
)

// validateNonEmpty returns a huh validation function that rejects blank input.
func validateNonEmpty(field string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s cannot be empty", field)
		}
		return nil
	}
}

// InitFormResult holds values collected by the init wizard.
type InitFormResult struct {
	MainBranch    string
	FeaturePrefix string
	HotfixPrefix  string
	TagPrefix     string
	DraftPR       bool
}

// InitFormResultFromDefaults creates an InitFormResult pre-filled from config defaults.
func InitFormResultFromDefaults(cfg config.Config) InitFormResult {
	return InitFormResult{
		MainBranch:    cfg.MainBranch,
		FeaturePrefix: cfg.FeaturePrefix,
		HotfixPrefix:  cfg.HotfixPrefix,
		TagPrefix:     cfg.TagPrefix,
		DraftPR:       cfg.DraftPROnStart,
	}
}

// ToPartialConfig converts the form result to a PartialConfig for saving.
func (r InitFormResult) ToPartialConfig() config.PartialConfig {
	draftPR := r.DraftPR
	return config.PartialConfig{
		MainBranch:     r.MainBranch,
		FeaturePrefix:  r.FeaturePrefix,
		HotfixPrefix:   r.HotfixPrefix,
		TagPrefix:      r.TagPrefix,
		DraftPROnStart: &draftPR,
	}
}

// RunInitForm displays the interactive init wizard and returns the user's choices.
func RunInitForm(defaults InitFormResult, branches []string) (InitFormResult, error) {
	result := defaults
	theme := DefaultTheme()

	branchOptions := make([]huh.Option[string], 0, len(branches))
	for _, b := range branches {
		branchOptions = append(branchOptions, huh.NewOption(b, b))
	}
	// If the default isn't in the detected list, prepend it
	found := false
	for _, b := range branches {
		if b == defaults.MainBranch {
			found = true
			break
		}
	}
	if !found {
		branchOptions = append([]huh.Option[string]{huh.NewOption(defaults.MainBranch, defaults.MainBranch)}, branchOptions...)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Main branch").
				Options(branchOptions...).
				Value(&result.MainBranch),
		).Title("Repository").Description("1/3"),

		huh.NewGroup(
			huh.NewInput().
				Title("Feature branch prefix").
				Value(&result.FeaturePrefix).
				Validate(validateNonEmpty("feature prefix")),
			huh.NewInput().
				Title("Hotfix branch prefix").
				Value(&result.HotfixPrefix).
				Validate(validateNonEmpty("hotfix prefix")),
		).Title("Branches").Description("2/3"),

		huh.NewGroup(
			huh.NewInput().
				Title("Tag prefix").
				Value(&result.TagPrefix).
				Validate(validateNonEmpty("tag prefix")),
			huh.NewConfirm().
				Title("Create draft PR on branch start?").
				Value(&result.DraftPR),
		).Title("Tags & PRs").Description("3/3"),
	).WithTheme(theme.HuhTheme())

	err := form.Run()
	if err != nil {
		return defaults, err
	}
	return result, nil
}

// InputPromptResult holds values from inline input prompts.
type InputPromptResult struct {
	Title string
	Body  string
}

// RunTitlePrompt shows an inline prompt for PR title and optional body.
func RunTitlePrompt(defaultTitle string, includeBody bool) (InputPromptResult, error) {
	result := InputPromptResult{Title: defaultTitle}
	theme := DefaultTheme()

	fields := []huh.Field{
		huh.NewInput().
			Title("PR title").
			Value(&result.Title).
			Validate(validateNonEmpty("PR title")),
	}
	if includeBody {
		fields = append(fields,
			huh.NewText().
				Title("PR body (optional)").
				Value(&result.Body),
		)
	}

	form := huh.NewForm(
		huh.NewGroup(fields...),
	).WithTheme(theme.HuhTheme())

	err := form.Run()
	if err != nil {
		return InputPromptResult{}, err
	}
	return result, nil
}

// RunMessagePrompt shows an inline prompt for a tag/release message.
func RunMessagePrompt(tagName string) (string, error) {
	var message string
	theme := DefaultTheme()

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Tag message for " + tagName).
				Description("Optional. Leave blank for a lightweight tag.").
				Value(&message),
		),
	).WithTheme(theme.HuhTheme())

	err := form.Run()
	if err != nil {
		return "", err
	}
	return message, nil
}
