package utils

import (
	"cmp"
	"regexp"
	"strings"
)

var (
	// suggested by https://semver.org/.
	semVerPattern = regexp.MustCompile(`^(?P<major>0|[1-9]\d*)\.(?P<minor>0|[1-9]\d*)\.(?P<patch>0|[1-9]\d*)(?:-(?P<prerelease>(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*)(?:\.(?:0|[1-9]\d*|\d*[a-zA-Z-][0-9a-zA-Z-]*))*))?(?:\+(?P<buildmetadata>[0-9a-zA-Z-]+(?:\.[0-9a-zA-Z-]+)*))?$`)
)

type SemVersion struct {
	raw           string
	Major         string
	Minor         string
	Patch         string
	PreRelease    string
	BuildMetaData string
}

func ParseVersion(version string) *SemVersion {
	semVer := &SemVersion{raw: version}

	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")

	matches := semVerPattern.FindStringSubmatch(version)
	for i, name := range semVerPattern.SubexpNames() {
		if i == 0 {
			continue
		} // subname start from 1.

		if i > len(matches) {
			break
		}

		switch name {
		case "major":
			semVer.Major = matches[i]
		case "minor":
			semVer.Minor = matches[i]
		case "patch":
			semVer.Patch = matches[i]
		case "prerelease":
			semVer.PreRelease = matches[i]
		case "buildmetadata":
			semVer.BuildMetaData = matches[i]
		}
	}

	return semVer
}

func (s *SemVersion) Raw() string {
	return s.raw
}

// Compare compare the main part ot a sematic version.
// Compare returns
//
// -1 if s is less than other,
// 0 if s equals other,
// +1 if s is greater than other.
func (s *SemVersion) Compare(other *SemVersion) int {
	cv := cmp.Compare(s.Major, other.Major)
	if cv != 0 {
		return cv
	}

	cv = cmp.Compare(s.Minor, other.Minor)
	if cv != 0 {
		return cv
	}

	cv = cmp.Compare(s.Patch, other.Patch)
	if cv != 0 {
		return cv
	}

	if s.PreRelease == "" && other.PreRelease != "" {
		return 1
	} else if s.PreRelease != "" && other.PreRelease == "" {
		return -1
	} else {

		if s.BuildMetaData == "" && other.BuildMetaData != "" {
			return 1
		} else if s.BuildMetaData != "" && other.BuildMetaData == "" {
			return -1
		}
		return 0
	}
}

func (s *SemVersion) Equal(other *SemVersion) bool {
	return s.raw == other.raw
}
