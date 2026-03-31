package version

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		input   string
		want    Version
		wantErr bool
	}{
		// Stable (existing cases, updated to named fields)
		{"1.2.3", Version{Major: 1, Minor: 2, Patch: 3}, false},
		{"0.0.0", Version{Major: 0, Minor: 0, Patch: 0}, false},
		{"10.20.30", Version{Major: 10, Minor: 20, Patch: 30}, false},
		{"v1.2.3", Version{Major: 1, Minor: 2, Patch: 3}, false},
		// Prerelease
		{"1.0.1-beta.1", Version{Major: 1, Minor: 0, Patch: 1, Prerelease: "beta", PreBuild: 1}, false},
		{"v1.0.1-beta.1", Version{Major: 1, Minor: 0, Patch: 1, Prerelease: "beta", PreBuild: 1}, false},
		{"2.3.4-rc.10", Version{Major: 2, Minor: 3, Patch: 4, Prerelease: "rc", PreBuild: 10}, false},
		{"0.1.0-alpha.1", Version{Major: 0, Minor: 1, Patch: 0, Prerelease: "alpha", PreBuild: 1}, false},
		// Invalid
		{"invalid", Version{}, true},
		{"1.2", Version{}, true},
		{"1.2.3.4", Version{}, true},
		{"1.0.1-beta", Version{}, true},    // missing counter
		{"1.0.1-.1", Version{}, true},      // empty suffix
		{"1.0.1-BETA.1", Version{}, true},  // uppercase
		{"1.0.1-be-ta.1", Version{}, true}, // dash in suffix
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

func TestString(t *testing.T) {
	tests := []struct {
		v    Version
		want string
	}{
		{Version{Major: 1, Minor: 2, Patch: 3}, "1.2.3"},
		{Version{Major: 1, Minor: 0, Patch: 1, Prerelease: "beta", PreBuild: 1}, "1.0.1-beta.1"},
		{Version{Major: 0, Minor: 1, Patch: 0, Prerelease: "rc", PreBuild: 5}, "0.1.0-rc.5"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.v.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsPrerelease(t *testing.T) {
	if (Version{Major: 1, Minor: 0, Patch: 0}).IsPrerelease() {
		t.Error("stable version should not be prerelease")
	}
	if !(Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "beta", PreBuild: 1}).IsPrerelease() {
		t.Error("prerelease version should be prerelease")
	}
}

func TestBump(t *testing.T) {
	v := Version{Major: 1, Minor: 2, Patch: 3}
	tests := []struct {
		scope string
		want  Version
	}{
		{"major", Version{Major: 2}},
		{"minor", Version{Major: 1, Minor: 3}},
		{"patch", Version{Major: 1, Minor: 2, Patch: 4}},
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

func TestBumpClearsPrerelease(t *testing.T) {
	v := Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "beta", PreBuild: 3}
	got, err := v.Bump("patch")
	if err != nil {
		t.Fatal(err)
	}
	if got.Prerelease != "" || got.PreBuild != 0 {
		t.Errorf("Bump should clear prerelease fields, got %v", got)
	}
	if got.Patch != 1 {
		t.Errorf("Bump(patch) on 1.0.0-beta.3 = %v, want 1.0.1", got)
	}
}

func TestBumpInvalidScope(t *testing.T) {
	v := Version{Major: 1, Minor: 2, Patch: 3}
	_, err := v.Bump("invalid")
	if err == nil {
		t.Error("expected error for invalid scope")
	}
}

func TestFormatWithPrefix(t *testing.T) {
	v := Version{Major: 1, Minor: 2, Patch: 3}
	if got := v.FormatWithPrefix("v"); got != "v1.2.3" {
		t.Errorf("FormatWithPrefix(\"v\") = %q, want %q", got, "v1.2.3")
	}
	pre := Version{Major: 1, Minor: 0, Patch: 1, Prerelease: "beta", PreBuild: 1}
	if got := pre.FormatWithPrefix("v"); got != "v1.0.1-beta.1" {
		t.Errorf("FormatWithPrefix(\"v\") = %q, want %q", got, "v1.0.1-beta.1")
	}
}

func TestLatest(t *testing.T) {
	versions := []Version{
		{Major: 1},
		{Major: 2, Minor: 1},
		{Major: 1, Minor: 5, Patch: 3},
		{Major: 2, Minor: 0, Patch: 9},
	}
	got := Latest(versions)
	want := Version{Major: 2, Minor: 1}
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

func TestLessThanPrerelease(t *testing.T) {
	tests := []struct {
		name string
		a, b Version
		want bool
	}{
		{
			"prerelease < stable same base",
			Version{Major: 1, Prerelease: "beta", PreBuild: 1},
			Version{Major: 1},
			true,
		},
		{
			"stable > prerelease same base",
			Version{Major: 1},
			Version{Major: 1, Prerelease: "beta", PreBuild: 1},
			false,
		},
		{
			"beta.1 < beta.2",
			Version{Major: 1, Prerelease: "beta", PreBuild: 1},
			Version{Major: 1, Prerelease: "beta", PreBuild: 2},
			true,
		},
		{
			"cross version: 1.0.0-beta.1 < 2.0.0-beta.1",
			Version{Major: 1, Prerelease: "beta", PreBuild: 1},
			Version{Major: 2, Prerelease: "beta", PreBuild: 1},
			true,
		},
		{
			"equal versions are not less than",
			Version{Major: 1, Minor: 2, Patch: 3},
			Version{Major: 1, Minor: 2, Patch: 3},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.LessThan(tt.b); got != tt.want {
				t.Errorf("%v.LessThan(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestBumpPrerelease(t *testing.T) {
	tests := []struct {
		name     string
		base     Version
		scope    string
		suffix   string
		existing []Version
		want     Version
	}{
		{
			"empty existing tags starts at 1",
			Version{Major: 1},
			"patch",
			"beta",
			nil,
			Version{Major: 1, Patch: 1, Prerelease: "beta", PreBuild: 1},
		},
		{
			"increments existing counter",
			Version{Major: 1},
			"patch",
			"beta",
			[]Version{
				{Major: 1, Patch: 1, Prerelease: "beta", PreBuild: 1},
				{Major: 1, Patch: 1, Prerelease: "beta", PreBuild: 2},
			},
			Version{Major: 1, Patch: 1, Prerelease: "beta", PreBuild: 3},
		},
		{
			"ignores tags for different target",
			Version{Major: 1},
			"patch",
			"beta",
			[]Version{
				{Major: 1, Minor: 1, Prerelease: "beta", PreBuild: 5},
			},
			Version{Major: 1, Patch: 1, Prerelease: "beta", PreBuild: 1},
		},
		{
			"ignores tags with different suffix",
			Version{Major: 1},
			"patch",
			"beta",
			[]Version{
				{Major: 1, Patch: 1, Prerelease: "rc", PreBuild: 3},
			},
			Version{Major: 1, Patch: 1, Prerelease: "beta", PreBuild: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.base.BumpPrerelease(tt.scope, tt.suffix, tt.existing)
			if err != nil {
				t.Fatalf("BumpPrerelease() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("BumpPrerelease() = %v, want %v", got, tt.want)
			}
		})
	}
}
