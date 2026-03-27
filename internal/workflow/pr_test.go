package workflow

import (
	"testing"

	"github.com/milis92/git-simple-flow/internal/ui"
)

func TestResolvePRInputPromptsOnlyForMissingTitleWhenBodyAlreadyProvided(t *testing.T) {
	u := &ui.UI{Interactive: true}
	called := false

	title, body, err := ResolvePRInput(
		u,
		func(defaultTitle string, includeBody bool) (ui.InputPromptResult, error) {
			called = true
			if defaultTitle != "Add auth" {
				t.Fatalf("defaultTitle = %q, want %q", defaultTitle, "Add auth")
			}
			if includeBody {
				t.Fatal("includeBody = true, want false when body was already provided")
			}
			return ui.InputPromptResult{Title: "Custom title", Body: "prompt body"}, nil
		},
		"feature/add-auth",
		"feature/",
		"",
		"existing body",
		true,
	)
	if err != nil {
		t.Fatalf("ResolvePRInput() error = %v", err)
	}
	if !called {
		t.Fatal("expected prompt to be called for missing title")
	}
	if title != "Custom title" {
		t.Fatalf("title = %q, want %q", title, "Custom title")
	}
	if body != "existing body" {
		t.Fatalf("body = %q, want existing provided body to be preserved", body)
	}
}

func TestResolvePRInputPromptsForMissingBody(t *testing.T) {
	u := &ui.UI{Interactive: true}

	title, body, err := ResolvePRInput(
		u,
		func(defaultTitle string, includeBody bool) (ui.InputPromptResult, error) {
			if defaultTitle != "Already set" {
				t.Fatalf("defaultTitle = %q, want %q", defaultTitle, "Already set")
			}
			if !includeBody {
				t.Fatal("includeBody = false, want true when body is missing")
			}
			return ui.InputPromptResult{Title: "Already set", Body: "prompt body"}, nil
		},
		"feature/add-auth",
		"feature/",
		"Already set",
		"",
		true,
	)
	if err != nil {
		t.Fatalf("ResolvePRInput() error = %v", err)
	}
	if title != "Already set" {
		t.Fatalf("title = %q, want %q", title, "Already set")
	}
	if body != "prompt body" {
		t.Fatalf("body = %q, want %q", body, "prompt body")
	}
}
