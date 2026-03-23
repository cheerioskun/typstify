package typst

import (
	"strconv"
	"strings"
)

// Version represents a semantic version number
type Version struct {
	Major int
	Minor int
	Patch int
}

// ParseVersion parses a version string like "v0.14.0" into a Version struct
func ParseVersion(s string) Version {
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")
	v := Version{}
	if len(parts) > 0 {
		v.Major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) > 1 {
		// Handle minor version with additional suffixes like "0.14.0-beta"
		minorPart := strings.Split(parts[1], "-")[0]
		v.Minor, _ = strconv.Atoi(minorPart)
	}
	if len(parts) > 2 {
		// Handle patch version with additional suffixes
		patchPart := strings.Split(parts[2], "-")[0]
		v.Patch, _ = strconv.Atoi(patchPart)
	}
	return v
}

// AtLeast checks if the version is at least the specified major.minor.patch
func (v Version) AtLeast(major, minor, patch int) bool {
	if v.Major > major {
		return true
	}
	if v.Major < major {
		return false
	}
	if v.Minor > minor {
		return true
	}
	if v.Minor < minor {
		return false
	}
	return v.Patch >= patch
}

// Current returns the current Typst version as a Version struct
func Current() Version {
	return ParseVersion(CurrentVersion())
}
