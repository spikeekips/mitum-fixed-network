package base

import (
	"bytes"
	"regexp"

	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

var (
	StringAddressType   = hint.Type("sas")
	StringAddressHint   = hint.NewHint(StringAddressType, "v0.0.1")
	StringAddressHinter = StringAddress{ht: StringAddressHint}
)

var (
	MaxAddressSize             = 100
	MinAddressSize             = AddressTypeSize + 3
	reBlankStringAddressString = regexp.MustCompile(`[\s][\s]*`)
	REStringAddressString      = `[a-zA-Z0-9][\w\-\.\!\$\*\@]*[a-zA-Z0-9]`
	reStringAddressString      = regexp.MustCompile(`^` + REStringAddressString + `$`)
)

type StringAddress struct {
	ht hint.Hint
	s  string
	b  [100]byte
}

func NewStringAddress(s string) StringAddress {
	return NewStringAddressWithHint(StringAddressHint, s)
}

func NewStringAddressWithHint(ht hint.Hint, s string) StringAddress {
	i := s + ht.Type().String()
	var b [100]byte
	copy(b[:], i)

	return StringAddress{ht: ht, s: i, b: b}
}

func MustNewStringAddress(s string) StringAddress {
	ad := NewStringAddress(s)

	if err := ad.IsValid(nil); err != nil {
		panic(err)
	}

	return ad
}

func ParseStringAddress(s string) (StringAddress, error) {
	p, ty, err := hint.ParseFixedTypedString(s, AddressTypeSize)
	switch {
	case err != nil:
		return StringAddress{}, err
	case ty != StringAddressType:
		return StringAddress{}, isvalid.InvalidError.Errorf("wrong hint of string address")
	}

	return NewStringAddress(p), nil
}

func (ad StringAddress) IsValid([]byte) error {
	if err := ad.ht.IsValid(nil); err != nil {
		return err
	}

	var b [100]byte
	copy(b[:], ad.s)

	if !bytes.Equal(ad.b[:], b[:]) {
		return isvalid.InvalidError.Errorf("wrong string address")
	}

	switch l := len(ad.s); {
	case l < MinAddressSize:
		return isvalid.InvalidError.Errorf("too short string address")
	case l > MaxAddressSize:
		return isvalid.InvalidError.Errorf("too long string address")
	}

	p := ad.s[:len(ad.s)-AddressTypeSize]
	if reBlankStringAddressString.MatchString(p) {
		return isvalid.InvalidError.Errorf("string address string, %q has blank", ad)
	}

	if !reStringAddressString.MatchString(p) {
		return isvalid.InvalidError.Errorf("invalid string address string, %q", ad)
	}

	switch {
	case len(ad.Hint().Type().String()) != AddressTypeSize:
		return isvalid.InvalidError.Errorf("wrong hint of string address")
	case ad.s[len(ad.s)-AddressTypeSize:] != ad.ht.Type().String():
		return isvalid.InvalidError.Errorf(
			"wrong type of string address; %v != %v", ad.s[len(ad.s)-AddressTypeSize:], ad.ht.Type())
	}

	return nil
}

func (ad StringAddress) Hint() hint.Hint {
	return ad.ht
}

func (ad StringAddress) SetHint(ht hint.Hint) hint.Hinter {
	if l := len(ad.s); l < MinAddressSize {
		ad.ht = ht

		return ad
	}

	return NewStringAddressWithHint(ht, ad.s[:len(ad.s)-AddressTypeSize])
}

func (ad StringAddress) String() string {
	return ad.s
}

func (ad StringAddress) Bytes() []byte {
	return []byte(ad.s)
}

func (ad StringAddress) Equal(b Address) bool {
	if b == nil {
		return false
	}

	if ad.Hint().Type() != b.Hint().Type() {
		return false
	}

	if err := b.IsValid(nil); err != nil {
		return false
	}

	return ad.s == b.String()
}
