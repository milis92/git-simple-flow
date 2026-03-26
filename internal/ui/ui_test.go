package ui

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

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
				Out: &bytes.Buffer{},
				In:  strings.NewReader(tt.input),
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
		Out: &bytes.Buffer{},
		In:  &failingReader{},
	}
	got, err := u.Confirm("Continue?")
	if got != false {
		t.Errorf("Confirm() = %v, want false", got)
	}
	if err != nil {
		t.Errorf("Confirm() error = %v, want nil (errors should be swallowed)", err)
	}
}

func TestConfirmPromptOutput(t *testing.T) {
	var buf bytes.Buffer
	u := &UI{
		Out: &buf,
		In:  strings.NewReader("n\n"),
	}
	_, _ = u.Confirm("Deploy?")
	if !strings.Contains(buf.String(), "Deploy?") {
		t.Errorf("prompt should contain message, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "[y/N]") {
		t.Errorf("prompt should contain [y/N], got %q", buf.String())
	}
}
