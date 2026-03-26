package ui

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestUIUsesThemeStyles(t *testing.T) {
	var buf bytes.Buffer
	u := &UI{
		Out:   &buf,
		In:    strings.NewReader(""),
		theme: DefaultTheme(),
	}

	u.Success("ok")
	output := buf.String()
	if !strings.Contains(output, "ok") {
		t.Errorf("Success output missing message, got %q", output)
	}
	if !strings.Contains(output, u.theme.IconDone) {
		t.Errorf("Success output should use theme icon %q, got %q", u.theme.IconDone, output)
	}
}

type failingReader struct{}

func (f *failingReader) Read([]byte) (int, error) {
	return 0, errors.New("broken pipe")
}

func TestConfirm(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    bool
		wantErr bool
	}{
		{"y confirms", "y\n", true, false},
		{"Y confirms", "Y\n", true, false},
		{"yes confirms", "yes\n", true, false},
		{"n declines", "n\n", false, false},
		{"no declines", "no\n", false, false},
		{"arbitrary input declines", "maybe\n", false, false},
		{"empty enter declines", "\n", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &UI{
				Out:   &bytes.Buffer{},
				In:    strings.NewReader(tt.input),
				theme: DefaultTheme(),
			}
			got, err := u.Confirm("Continue?")
			if (err != nil) != tt.wantErr {
				t.Errorf("Confirm() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Confirm() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfirmBrokenReader(t *testing.T) {
	u := &UI{
		Out:   &bytes.Buffer{},
		In:    &failingReader{},
		theme: DefaultTheme(),
	}
	got, err := u.Confirm("Continue?")
	if got != false {
		t.Errorf("Confirm() = %v, want false", got)
	}
	if err == nil {
		t.Error("Confirm() error = nil, want error for broken reader")
	}
}

func TestConfirmPromptOutput(t *testing.T) {
	var buf bytes.Buffer
	u := &UI{
		Out:   &buf,
		In:    strings.NewReader("n\n"),
		theme: DefaultTheme(),
	}
	_, _ = u.Confirm("Deploy?")
	if !strings.Contains(buf.String(), "Deploy?") {
		t.Errorf("prompt should contain message, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "[y/N]") {
		t.Errorf("prompt should contain [y/N], got %q", buf.String())
	}
}

func TestConfirmAutoConfirm(t *testing.T) {
	var buf bytes.Buffer
	u := &UI{
		Out:         &buf,
		In:          strings.NewReader(""),
		AutoConfirm: true,
		theme:       DefaultTheme(),
	}
	got, err := u.Confirm("Deploy?")
	if err != nil {
		t.Errorf("Confirm() error = %v, want nil", err)
	}
	if !got {
		t.Error("Confirm() = false, want true when AutoConfirm is set")
	}
	if !strings.Contains(buf.String(), "auto-confirmed") {
		t.Errorf("output should contain 'auto-confirmed', got %q", buf.String())
	}
}

func TestShouldPrompt(t *testing.T) {
	tests := []struct {
		name string
		ui   UI
		want bool
	}{
		{
			name: "interactive prompt allowed",
			ui: UI{
				Interactive: true,
				AutoConfirm: false,
			},
			want: true,
		},
		{
			name: "auto confirm disables optional prompts",
			ui: UI{
				Interactive: true,
				AutoConfirm: true,
			},
			want: false,
		},
		{
			name: "non interactive disables prompts",
			ui: UI{
				Interactive: false,
				AutoConfirm: false,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ui.ShouldPrompt(); got != tt.want {
				t.Errorf("ShouldPrompt() = %v, want %v", got, tt.want)
			}
		})
	}
}
