package version

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major int
	Minor int
	Patch int
}

func Parse(s string) (Version, error) {
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")
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
	return Version{Major: major, Minor: minor, Patch: patch}, nil
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v Version) Bump(scope string) (Version, error) {
	switch scope {
	case "major":
		return Version{Major: v.Major + 1, Minor: 0, Patch: 0}, nil
	case "minor":
		return Version{Major: v.Major, Minor: v.Minor + 1, Patch: 0}, nil
	case "patch":
		return Version{Major: v.Major, Minor: v.Minor, Patch: v.Patch + 1}, nil
	default:
		return Version{}, fmt.Errorf("invalid scope: %q (must be major, minor, or patch)", scope)
	}
}

func (v Version) FormatWithPrefix(prefix string) string {
	return prefix + v.String()
}

func (v Version) LessThan(other Version) bool {
	if v.Major != other.Major {
		return v.Major < other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor < other.Minor
	}
	return v.Patch < other.Patch
}

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
