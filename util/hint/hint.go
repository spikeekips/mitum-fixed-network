package hint

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"regexp"

	"golang.org/x/mod/semver"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
)

const (
	MaxVersionSize          int    = 15
	MaxHintSize             int    = MaxVersionSize + 2
	HintVerboseFormat       string = `hint{type=%q code="%x" version=%q}`
	HintMarshalStringFormat string = "%x+%s"
)

var (
	ReHintMarshalStringFormat                = `(?P<type>[a-f0-9]{4})\+(?P<version>.*)`
	reHintMarshalString       *regexp.Regexp = regexp.MustCompile("^" + ReHintMarshalStringFormat + "$")
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
	if len(s) > MaxHintSize+1 {
		return Hint{}, xerrors.Errorf("wrong string for Hint; len=%d", len(s))
	}

	if !reHintMarshalString.MatchString(s) {
		return Hint{}, xerrors.Errorf("unknown format of hint: %q", s)
	}

	ms := reHintMarshalString.FindStringSubmatch(s)
	if len(ms) != 3 {
		return Hint{}, xerrors.Errorf("something empty of hint: %q", s)
	}

	var code [2]byte
	if b, err := hex.DecodeString(ms[1]); err != nil {
		return Hint{}, err
	} else {
		code = [2]byte{b[0], b[1]}
	}

	ht := Hint{t: Type(code), version: util.Version(ms[2])}

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
	return fmt.Sprintf(
		HintVerboseFormat,
		ht.Type().String(),
		[2]byte(ht.Type()),
		ht.version,
	)
}

func (ht Hint) String() string {
	return fmt.Sprintf(
		HintMarshalStringFormat,
		[2]byte(ht.Type()),
		ht.version,
	)
}

type Hinter interface {
	Hint() Hint
}
