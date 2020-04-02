package hint

import (
	"strings"

	"golang.org/x/mod/semver"

	"github.com/spikeekips/mitum/errors"
)

var (
	InvalidVersionError       = errors.NewError("invalid version found")
	VersionDoesNotMatchError  = errors.NewError("version does not match")
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

// IsCompatible checks if the check version is compatible to the target. The
// compatible conditions are,
// - major matches
// - minor of the check version is same or lower than target
// - patch of the check version is same or lower than target
func (vs Version) IsCompatible(check Version) error {
	a := vs.GO()
	b := check.GO()

	if !semver.IsValid(a) {
		return InvalidVersionError.Wrapf("version=%s", a)
	}
	if !semver.IsValid(b) {
		return InvalidVersionError.Wrapf("version=%s", b)
	}

	if semver.Major(a) != semver.Major(b) {
		return VersionNotCompatibleError.Wrapf("target=%s != check=%s", semver.Major(a), semver.Major(b))
	}
	if semver.Compare(semver.MajorMinor(a), semver.MajorMinor(b)) < 0 {
		return VersionNotCompatibleError.Wrapf("target=%s < check=%s", semver.MajorMinor(a), semver.MajorMinor(b))
	}
	if semver.Compare(a, b) < 0 {
		return VersionNotCompatibleError.Wrapf("target=%s < check=%s", a, b)
	}

	return nil
}
