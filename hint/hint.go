package hint

import (
	"bytes"
	"fmt"

	"golang.org/x/mod/semver"
	"golang.org/x/xerrors"
)

const (
	MaxVersionSize int = 15
	MaxHintSize    int = MaxVersionSize + 2
)

type Hint struct {
	t       Type
	version Version
}

func NewHint(t Type, version Version) (Hint, error) {
	ht := Hint{t: t, version: version}

	return ht, ht.IsValid(nil)
}

func MustHint(t Type, version Version) Hint {
	ht := Hint{t: t, version: version}
	if err := ht.IsValid(nil); err != nil {
		panic(err)
	}

	return ht
}

func NewHintFromBytes(b []byte) (Hint, error) {
	if len(b) > MaxHintSize {
		return Hint{}, xerrors.Errorf("wrong bytes for Hint; len=%d", len(b))
	}

	var t [2]byte
	_ = copy(t[:], b[:2])

	ht := Hint{
		t:       Type(t),
		version: Version(bytes.TrimRight(b[2:], "\x00")),
	}

	return ht, ht.IsValid(nil)
}

func (ht Hint) IsValid([]byte) error {
	if err := ht.Type().IsValid(nil); err != nil {
		return err
	}

	if len(ht.version) > MaxVersionSize {
		return InvalidVersionError.Wrapf("oversized version; len=%d", len(ht.version))
	} else if !semver.IsValid(ht.version.GO()) {
		return InvalidVersionError.Wrapf("version=%s", ht.version)
	}

	return nil
}

func (ht Hint) IsRegistered() error {
	if !IsRegisteredType(ht.Type()) {
		return NotRegisteredTypeFoundError.Wrapf("type=%s", ht.Verbose())
	}

	return nil
}

func (ht Hint) Type() Type {
	return ht.t
}

func (ht Hint) Version() Version {
	return ht.version
}

func (ht Hint) Equal(h Hint) bool {
	if !ht.Type().Equal(h.Type()) {
		return false
	}

	if ht.Version() != h.Version() {
		return false
	}

	return true
}

func (ht Hint) IsCompatible(check Hint) error {
	if !ht.Type().Equal(check.Type()) {
		return NewTypeDoesNotMatchError(ht.Type(), check.Type())
	}

	return ht.Version().IsCompatible(check.Version())
}

func (ht Hint) Bytes() []byte {
	b := bytes.NewBuffer(ht.t[:])
	_, _ = b.Write([]byte(ht.version))

	return b.Bytes()
}

func (ht Hint) Verbose() string {
	return fmt.Sprintf("type=%s version=%s", ht.Type().Verbose(), ht.version)
}

type Hinter interface {
	Hint() Hint
}

type IsHinted interface {
	IsHinted() bool
}
