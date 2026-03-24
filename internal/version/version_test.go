package version

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		input   string
		want    Version
		wantErr bool
	}{
		{"1.2.3", Version{1, 2, 3}, false},
		{"0.0.0", Version{0, 0, 0}, false},
		{"10.20.30", Version{10, 20, 30}, false},
		{"v1.2.3", Version{1, 2, 3}, false},
		{"invalid", Version{}, true},
		{"1.2", Version{}, true},
		{"1.2.3.4", Version{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBump(t *testing.T) {
	v := Version{1, 2, 3}
	tests := []struct {
		scope string
		want  Version
	}{
		{"major", Version{2, 0, 0}},
		{"minor", Version{1, 3, 0}},
		{"patch", Version{1, 2, 4}},
	}
	for _, tt := range tests {
		t.Run(tt.scope, func(t *testing.T) {
			got, err := v.Bump(tt.scope)
			if err != nil {
				t.Fatalf("Bump(%q) error: %v", tt.scope, err)
			}
			if got != tt.want {
				t.Errorf("Bump(%q) = %v, want %v", tt.scope, got, tt.want)
			}
		})
	}
}

func TestBumpInvalidScope(t *testing.T) {
	v := Version{1, 2, 3}
	_, err := v.Bump("invalid")
	if err == nil {
		t.Error("expected error for invalid scope")
	}
}

func TestFormatWithPrefix(t *testing.T) {
	v := Version{1, 2, 3}
	if got := v.FormatWithPrefix("v"); got != "v1.2.3" {
		t.Errorf("FormatWithPrefix(\"v\") = %q, want %q", got, "v1.2.3")
	}
	if got := v.FormatWithPrefix("release-"); got != "release-1.2.3" {
		t.Errorf("FormatWithPrefix(\"release-\") = %q, want %q", got, "release-1.2.3")
	}
}

func TestLatest(t *testing.T) {
	versions := []Version{
		{1, 0, 0},
		{2, 1, 0},
		{1, 5, 3},
		{2, 0, 9},
	}
	got := Latest(versions)
	want := Version{2, 1, 0}
	if got != want {
		t.Errorf("Latest() = %v, want %v", got, want)
	}
}

func TestLatestEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on empty slice")
		}
	}()
	Latest([]Version{})
}
