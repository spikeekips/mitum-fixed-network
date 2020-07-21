package hint

import (
	"fmt"
	"strings"

	"github.com/spikeekips/mitum/util/errors"
)

var (
	InvalidTypeError             = errors.NewError("invalid Type")
	UnknownTypeError             = errors.NewError("unknown Type")
	TypeAlreadyRegisteredError   = errors.NewError("Type already registered")
	DuplicatedTypeNameFoundError = errors.NewError("same Type name already registered")
	TypeDoesNotMatchError        = errors.NewError("type does not match")
)

var NullType = Type{}
var TypeVerboseFormat string = `type{name=%q code=%q}`

// NOTE typeNames and nameTypes maintain all the registered Type and it's name.
var (
	typeNames map[Type]string
	nameTypes map[string]Type
)

func init() {
	typeNames = map[Type]string{}
	nameTypes = map[string]Type{}
}

// Type represents the type of struct, or any arbitrary data. Type defines the
// type of object. It should be unique thru the runtime.
//
// - 0x00 ~ 0x10 range is for mitum itself.
// - 0xff,- range is reserved for testing.
type Type [2]byte

func MustNewType(a, b byte, name string) Type {
	t := Type{a, b}
	if err := registerType(t, name); err != nil {
		panic(err)
	}

	return t
}

// Name returns the name of Type.
func (ty Type) Name() string {
	if _, found := typeNames[ty]; !found {
		return ""
	}

	return typeNames[ty]
}

// String returns the byte strings of Type.
func (ty Type) String() string {
	if _, found := typeNames[ty]; !found {
		return ""
	}

	return fmt.Sprintf("%x", ty[:])
}

// IsValid checks Type
func (ty Type) IsValid([]byte) error {
	if ty == NullType {
		return InvalidTypeError.Errorf("empty Type")
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
	return fmt.Sprintf(TypeVerboseFormat, ty.Name(), ty.String())
}

func isRegisteredType(t Type) bool {
	_, found := typeNames[t]
	return found
}

// registerType registers the givven Type in globals
func registerType(t Type, name string) error {
	if err := t.IsValid(nil); err != nil {
		return err
	}

	name = strings.TrimSpace(name)

	if _, found := typeNames[t]; found {
		return TypeAlreadyRegisteredError.Errorf("type=%s", t.Verbose())
	} else if _, found := nameTypes[name]; found {
		return DuplicatedTypeNameFoundError.Errorf("type=%s", t.Verbose())
	}

	typeNames[t] = name
	nameTypes[name] = t

	return nil
}

func NewTypeDoesNotMatchError(target, check Type) error {
	return TypeDoesNotMatchError.Errorf("target=%s != check=%s", target.Verbose(), check.Verbose())
}
