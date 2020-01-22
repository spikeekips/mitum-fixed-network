package hint

import (
	"fmt"
	"strings"

	"github.com/spikeekips/mitum/errors"
)

var (
	InvalidTypeError             = errors.NewError("invalid Type")
	NotRegisteredTypeFoundError  = errors.NewError("unknown Type")
	TypeAlreadyRegisteredError   = errors.NewError("Type already registered")
	DuplicatedTypeNameFoundError = errors.NewError("same Type name already registered")
	TypeDoesNotMatchError        = errors.NewError("type does not match")
)

var NullType = Type{}

// NOTE typeNames and nameTypes maintain all the registered Type and it's name.
var typeNames map[Type]string
var nameTypes map[string]Type

func init() {
	typeNames = map[Type]string{}
	nameTypes = map[string]Type{}
}

// Type represents the type of struct, or any arbitrary data.
// NOTE '0xff' of first element of Type is reserved for testing.
type Type [2]byte

// String returns the name of Type.
func (ty Type) String() string {
	if _, found := typeNames[ty]; !found {
		return ""
	}

	return typeNames[ty]
}

// IsValid checks Type
func (ty Type) IsValid([]byte) error {
	if ty == NullType {
		return InvalidTypeError.Wrapf("empty Type")
	}

	return nil
}

// Equal checks 2 types are same
func (ty Type) Equal(t Type) bool {
	return ty == t
}

// Bytes returns [2]byte
func (ty Type) Bytes() []byte {
	return ty[:]
}

// Verbose shows the detailed Type info
func (ty Type) Verbose() string {
	return fmt.Sprintf("%v(%s)", [2]byte(ty), ty)
}

func IsRegisteredType(t Type) bool {
	_, found := typeNames[t]
	return found
}

// IsRegisteredTypeName checks the given name is registered or not
func IsRegisteredTypeName(name string) bool {
	_, found := nameTypes[name]
	return found
}

// RegisterType registers the givven Type in globals
func RegisterType(t Type, name string) error {
	if err := t.IsValid(nil); err != nil {
		return err
	}

	name = strings.TrimSpace(name)

	if _, found := typeNames[t]; found {
		return TypeAlreadyRegisteredError.Wrapf("type=%s", t.Verbose())
	} else if _, found := nameTypes[name]; found {
		return DuplicatedTypeNameFoundError.Wrapf("type=%s name=%s", t.Verbose(), name)
	}

	typeNames[t] = name
	nameTypes[name] = t

	return nil
}

// typeNames returns the name of the given Type
func TypeByName(name string) (Type, error) {
	t, found := nameTypes[name]
	if !found {
		return Type{}, NotRegisteredTypeFoundError.Wrapf("no Type found; name=%s", name)
	}

	return t, nil
}

func NewTypeDoesNotMatchError(target, check Type) error {
	return TypeDoesNotMatchError.Wrapf("target=%s != check=%s", target.Verbose(), check.Verbose())
}
