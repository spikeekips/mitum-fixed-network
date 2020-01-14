package hint

import (
	"strings"

	"github.com/spikeekips/mitum/errors"
	"golang.org/x/mod/semver"
)

var (
	InvalidVersionError       = errors.NewError("invalid version found")
	VersionDoesNotMatchError  = errors.NewError("version does not match")
	VersionNotCompatibleError = errors.NewError("versions not compatible")
)

// VersionGO returns golang style semver string. It does not check IsValid().
func VersionGO(version string) string {
	if strings.HasPrefix(version, "v") {
		return version
	}

	return "v" + version
}

// IsVersionCompatible checks if the check version is compatible to the target.
// The compatible conditions are,
// - major matches
// - minor of the check version is same or lower than target
// - patch of the check version is same or lower than target
func IsVersionCompatible(target, check string) error {
	a := VersionGO(target)
	b := VersionGO(check)
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
