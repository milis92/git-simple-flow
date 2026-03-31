// Package version provides semantic versioning (semver) parsing, comparison, and bumping.
package version

import (
	"fmt"
	"strconv"
	"strings"
)

// Version represents a semantic version with major, minor, and patch components,
// plus optional prerelease suffix and counter (e.g. "beta.1").
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string // "beta", "rc", "alpha" — empty for stable
	PreBuild   int    // 1, 2, 3 — counter after suffix
}

// Parse parses a semver string into a Version. The "v" prefix is optional
// and will be stripped if present. Accepts both stable ("1.2.3") and
// prerelease ("1.2.3-beta.1") formats. Prerelease suffix must be lowercase
// alphanumeric with a required numeric counter.
func Parse(s string) (Version, error) {
	s = strings.TrimPrefix(s, "v")

	// Split base from prerelease on first "-"
	base, pre, _ := strings.Cut(s, "-")

	parts := strings.Split(base, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid semver: %q", s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %q", parts[0])
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version: %q", parts[1])
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version: %q", parts[2])
	}

	v := Version{Major: major, Minor: minor, Patch: patch}

	if pre == "" {
		return v, nil
	}

	// Parse prerelease: must be "suffix.counter"
	preParts := strings.SplitN(pre, ".", 2)
	if len(preParts) != 2 || preParts[0] == "" {
		return Version{}, fmt.Errorf("invalid prerelease format: %q (expected suffix.counter)", pre)
	}

	for _, c := range preParts[0] {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return Version{}, fmt.Errorf("invalid prerelease suffix %q: must be lowercase alphanumeric", preParts[0])
		}
	}

	counter, err := strconv.Atoi(preParts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid prerelease counter: %q", preParts[1])
	}

	v.Prerelease = preParts[0]
	v.PreBuild = counter
	return v, nil
}

// String formats the version. Stable: "1.2.3". Prerelease: "1.2.3-beta.1".
func (v Version) String() string {
	if v.Prerelease != "" {
		return fmt.Sprintf("%d.%d.%d-%s.%d", v.Major, v.Minor, v.Patch, v.Prerelease, v.PreBuild)
	}
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// IsPrerelease reports whether this version has a prerelease suffix.
func (v Version) IsPrerelease() bool {
	return v.Prerelease != ""
}

// Bump returns a new Version with the given scope incremented.
// Valid scopes are "major", "minor", and "patch". Higher components
// reset lower ones to zero (e.g. bumping minor resets patch).
// Prerelease fields are always cleared.
// Returns an error for an invalid scope.
func (v Version) Bump(scope string) (Version, error) {
	switch scope {
	case "major":
		return Version{Major: v.Major + 1}, nil
	case "minor":
		return Version{Major: v.Major, Minor: v.Minor + 1}, nil
	case "patch":
		return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}, nil
	default:
		return Version{}, fmt.Errorf("invalid scope: %q (must be major, minor, or patch)", scope)
	}
}

// FormatWithPrefix returns the version string with the given prefix prepended
// (e.g. FormatWithPrefix("v") returns "v1.2.3" or "v1.2.3-beta.1").
func (v Version) FormatWithPrefix(prefix string) string {
	return prefix + v.String()
}

// LessThan reports whether v is strictly less than other, following semver
// precedence: major, minor, patch, then prerelease < stable for the same
// base version, then prerelease counter.
func (v Version) LessThan(other Version) bool {
	if v.Major != other.Major {
		return v.Major < other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor < other.Minor
	}
	if v.Patch != other.Patch {
		return v.Patch < other.Patch
	}
	// Same base version: prerelease < stable
	if v.IsPrerelease() != other.IsPrerelease() {
		return v.IsPrerelease()
	}
	// Both prerelease (or both stable): compare counter
	return v.PreBuild < other.PreBuild
}

// BumpPrerelease computes the next prerelease version. The receiver is the
// latest stable version. It bumps by scope to find the target stable version,
// then finds the highest counter for that target+suffix in existingTags and
// increments it (or starts at 1).
func (v Version) BumpPrerelease(scope, suffix string, existingTags []Version) (Version, error) {
	target, err := v.Bump(scope)
	if err != nil {
		return Version{}, err
	}

	maxCounter := 0
	for _, t := range existingTags {
		if t.Major == target.Major && t.Minor == target.Minor && t.Patch == target.Patch &&
			t.Prerelease == suffix {
			if t.PreBuild > maxCounter {
				maxCounter = t.PreBuild
			}
		}
	}

	return Version{
		Major:      target.Major,
		Minor:      target.Minor,
		Patch:      target.Patch,
		Prerelease: suffix,
		PreBuild:   maxCounter + 1,
	}, nil
}

// Latest returns the highest version from the given slice.
// It panics if the slice is empty.
func Latest(versions []Version) Version {
	if len(versions) == 0 {
		panic("no versions provided")
	}
	latest := versions[0]
	for _, v := range versions[1:] {
		if latest.LessThan(v) {
			latest = v
		}
	}
	return latest
}
