package hint

import (
	"bytes"
	"fmt"

	"golang.org/x/mod/semver"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
)

const (
	MaxVersionSize   int    = 15
	MaxHintSize      int    = MaxVersionSize + 2
	HintStringFormat string = `hint{type=%q code="%x" version=%q}`
)

type Hint struct {
	t       Type
	version util.Version
}

func NewHint(t Type, version util.Version) (Hint, error) {
	ht := Hint{t: t, version: version}

	return ht, ht.IsValid(nil)
}

func MustHint(t Type, version util.Version) Hint {
	ht := Hint{t: t, version: version}
	if err := ht.IsValid(nil); err != nil {
		panic(err)
	}

	return ht
}

func NewHintFromString(s string) (Hint, error) {
	var name, version string
	var code []byte
	n, err := fmt.Sscanf(s, HintStringFormat, &name, &code, &version)
	if err != nil {
		return Hint{}, err
	}
	if n != 3 {
		return Hint{}, xerrors.Errorf("invalid formatted hint string found: hint=%q", s)
	}
	if len(code) != 2 {
		return Hint{}, xerrors.Errorf("invalid formatted hint code found: hint=%q", s)
	}

	ht := Hint{t: Type([2]byte{code[0], code[1]}), version: util.Version(version)}

	return ht, ht.IsValid(nil)
}

func NewHintFromBytes(b []byte) (Hint, error) {
	if len(b) > MaxHintSize {
		return Hint{}, xerrors.Errorf("wrong bytes for Hint; len=%d", len(b))
	}

	var t [2]byte
	_ = copy(t[:], b[:2])

	ht := Hint{
		t:       Type(t),
		version: util.Version(bytes.TrimRight(b[2:], "\x00")),
	}

	return ht, ht.IsValid(nil)
}

func (ht Hint) IsValid([]byte) error {
	if err := ht.Type().IsValid(nil); err != nil {
		return err
	}

	if len(ht.version) > MaxVersionSize {
		return util.InvalidVersionError.Errorf("oversized version; len=%d", len(ht.version))
	} else if !semver.IsValid(ht.version.GO()) {
		return util.InvalidVersionError.Errorf("version=%s", ht.version)
	}

	return nil
}

func (ht Hint) IsRegistered() error {
	if !isRegisteredType(ht.Type()) {
		return UnknownTypeError.Errorf("hint=%s", ht.Verbose())
	}

	return nil
}

func (ht Hint) Type() Type {
	return ht.t
}

func (ht Hint) Version() util.Version {
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
	return fmt.Sprintf("%s(%s)", ht.Type().Verbose(), ht.version)
}

func (ht Hint) String() string {
	return fmt.Sprintf(
		HintStringFormat,
		ht.Type().String(),
		[2]byte(ht.Type()),
		ht.version,
	)
}

type Hinter interface {
	Hint() Hint
}
