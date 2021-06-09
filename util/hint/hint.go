package hint

import (
	"fmt"
	"strings"

	"github.com/spikeekips/mitum/util/isvalid"
	"golang.org/x/mod/semver"
	"golang.org/x/xerrors"
)

var (
	MaxVersionLength int = 20
	MaxHintLength    int = MaxTypeLength + MaxVersionLength + 1
)

type Hinter interface {
	Hint() Hint
}

type Hint struct {
	ty Type
	v  string
}

func NewHint(ty Type, v string) Hint {
	return Hint{ty: ty, v: v}
}

// ParseHint parses string and returns Hint; it does not do valid
// check(IsValid()).
func ParseHint(s string) (Hint, error) {
	switch i := strings.Split(s, "-v"); {
	case len(i) < 2:
		return Hint{}, isvalid.InvalidError.Errorf("invalid Hint format found, %q", s)
	default:
		return NewHint(Type(strings.Join(i[:len(i)-1], "-v")), "v"+i[len(i)-1]), nil
	}
}

func (ht Hint) IsValid([]byte) error {
	if err := ht.ty.IsValid(nil); err != nil {
		return xerrors.Errorf("invalid Hint: %w", err)
	} else if len(ht.v) > MaxVersionLength {
		return isvalid.InvalidError.Errorf("version too long, %d", MaxVersionLength)
	}

	if !semver.IsValid(ht.v) {
		return isvalid.InvalidError.Errorf("invalid version, %q", ht.v)
	}

	return nil
}

func (ht Hint) Type() Type {
	return ht.ty
}

func (ht Hint) Version() string {
	return ht.v
}

func (ht Hint) Equal(b Hint) bool {
	return ht.ty == b.ty && ht.v == b.v
}

// IsCompatible checks whether target is compatible with source, ht.
// - Obviously, Type should be same
// - If same version, compatible
// - If major version is different, not compatible
// - If same major, but minor version of target is lower than source, not compatible
// - If same major and minor, but different patch version, compatible
func (ht Hint) IsCompatible(target Hint) error {
	if ht.ty != target.ty {
		return xerrors.Errorf("type does not match; %q != %q", ht.ty, target.ty)
	}

	switch {
	case semver.Major(ht.v) != semver.Major(target.v):
		return xerrors.Errorf("not compatible; different major version")
	case semver.Compare(semver.MajorMinor(ht.v), semver.MajorMinor(target.v)) < 0:
		return xerrors.Errorf("not compatible; lower minor version")
	default:
		return nil
	}
}

func (ht Hint) Bytes() []byte {
	return []byte(ht.String())
}

func (ht Hint) String() string {
	if len(ht.ty) < 1 && len(ht.v) < 1 {
		return ""
	}

	return fmt.Sprintf("%s-%s", ht.ty, ht.v)
}

func (ht Hint) MarshalText() ([]byte, error) {
	return []byte(ht.String()), nil
}

func (ht *Hint) UnmarshalText(b []byte) error {
	if len(b) < 1 {
		return nil
	}

	if i, err := ParseHint(string(b)); err != nil {
		return err
	} else {
		ht.ty = i.ty
		ht.v = i.v

		return nil
	}
}
