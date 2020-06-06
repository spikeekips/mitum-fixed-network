package util

import (
	"strings"

	"golang.org/x/mod/semver"

	"github.com/spikeekips/mitum/util/errors"
)

var (
	InvalidVersionError       = errors.NewError("invalid version found")
	VersionNotCompatibleError = errors.NewError("versions not compatible")
)

type Version string

func (vs Version) String() string {
	return string(vs)
}

// GO returns golang style semver string. It does not check IsValid().
func (vs Version) GO() string {
	s := string(vs)
	if strings.HasPrefix(s, "v") {
		return s
	}

	return "v" + s
}

func (vs Version) IsValid([]byte) error {
	if !semver.IsValid(vs.GO()) {
		return InvalidVersionError.Errorf("version=%s", vs)
	}

	return nil
}

// IsCompatible checks if the check version is compatible to the target. The
// compatible conditions are,
// - major matches
// - minor of the check version is same or lower than target
// - patch of the check version is same or lower than target
func (vs Version) IsCompatible(check Version) error {
	a := vs.GO()
	b := check.GO()

	if !semver.IsValid(a) {
		return InvalidVersionError.Errorf("version=%s", a)
	}
	if !semver.IsValid(b) {
		return InvalidVersionError.Errorf("version=%s", b)
	}

	if semver.Major(a) != semver.Major(b) {
		return VersionNotCompatibleError.Errorf("target=%s != check=%s", semver.Major(a), semver.Major(b))
	}
	if semver.Compare(semver.MajorMinor(a), semver.MajorMinor(b)) < 0 {
		return VersionNotCompatibleError.Errorf("target=%s < check=%s", semver.MajorMinor(a), semver.MajorMinor(b))
	}
	if semver.Compare(a, b) < 0 {
		return VersionNotCompatibleError.Errorf("target=%s < check=%s", a, b)
	}

	return nil
}
