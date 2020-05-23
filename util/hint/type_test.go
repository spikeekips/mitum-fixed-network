package hint

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

// isRegisteredTypeName checks the given name is registered or not
func isRegisteredTypeName(name string) bool {
	_, found := nameTypes[name]
	return found
}

type testType struct {
	suite.Suite
}

func (t *testType) TestNew() {
	ty := Type{0x00, 0xff}
	t.IsType(Type{}, ty)
}

func (t *testType) TestIsValid() {
	ty := Type{0x00, 0xff}
	t.NoError(ty.IsValid(nil))

	ty = Type{0x00, 0x00}
	t.True(xerrors.Is(ty.IsValid(nil), InvalidTypeError))

	ty = Type{}
	t.True(xerrors.Is(ty.IsValid(nil), InvalidTypeError))
}

func (t *testType) TestRegister() {
	tn := "showme-ff-00"
	ty := Type{0xff, 0x00}
	t.False(isRegisteredType(ty))
	t.False(isRegisteredTypeName(tn))

	err := registerType(ty, tn)
	t.NoError(err)
	t.True(isRegisteredType(ty))
	t.True(isRegisteredTypeName(tn))

	// register again
	err = registerType(ty, tn)
	t.True(xerrors.Is(err, TypeAlreadyRegisteredError))
}

func (t *testType) TestString() {
	tn := "showme-ff-01"
	ty := Type{0xff, 0x01}

	_ = registerType(ty, tn)
	t.Equal(tn, ty.String())
}

func (t *testType) TestRegisterSameName() {
	tn := "showme-ff-02"
	{
		ty := Type{0xff, 0x02}
		_ = registerType(ty, tn)
		t.True(isRegisteredType(ty))
	}

	ty := Type{0xff, 0x03}
	err := registerType(ty, tn)
	t.True(xerrors.Is(err, DuplicatedTypeNameFoundError))
}

func TestType(t *testing.T) {
	suite.Run(t, new(testType))
}

func (t *testType) TestMarshal() {
	ty := Type{0xff, 0x04}
	_ = registerType(ty, "0xff03-showme")

	b, err := json.Marshal(ty)
	t.NoError(err)

	var unmarshaled Type
	t.NoError(json.Unmarshal(b, &unmarshaled))
}
