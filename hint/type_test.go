package hint

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testType struct {
	suite.Suite
}

func (t *testType) TestNew() {
	ty := Type([2]byte{0x00, 0xff})
	t.IsType(Type{}, ty)
}

func (t *testType) TestIsValid() {
	ty := Type([2]byte{0x00, 0xff})
	t.NoError(ty.IsValid())

	ty = Type([2]byte{0x00, 0x00})
	t.True(xerrors.Is(ty.IsValid(), InvalidTypeError))

	ty = Type{}
	t.True(xerrors.Is(ty.IsValid(), InvalidTypeError))
}

func (t *testType) TestRegister() {
	tn := "showme-ff-00"
	ty := Type([2]byte{0xff, 0x00})
	t.False(IsRegisteredType(ty))
	t.False(IsRegisteredTypeName(tn))

	err := RegisterType(ty, tn)
	t.NoError(err)
	t.True(IsRegisteredType(ty))
	t.True(IsRegisteredTypeName(tn))

	// register again
	err = RegisterType(ty, tn)
	t.True(xerrors.Is(err, TypeAlreadyRegisteredError))
}

func (t *testType) TestString() {
	tn := "showme-ff-01"
	ty := Type([2]byte{0xff, 0x01})

	_ = RegisterType(ty, tn)
	t.Equal(tn, ty.String())
}

func (t *testType) TestRegisterSameName() {
	tn := "showme-ff-02"
	{
		ty := Type([2]byte{0xff, 0x02})
		_ = RegisterType(ty, tn)
		t.True(IsRegisteredType(ty))
	}

	ty := Type([2]byte{0xff, 0x03})
	err := RegisterType(ty, tn)
	t.True(xerrors.Is(err, DuplicatedTypeNameFoundError))
}

func TestType(t *testing.T) {
	suite.Run(t, new(testType))
}

func (t *testType) TestMarshal() {
	ty := Type([2]byte{0xff, 0x04})
	_ = RegisterType(ty, "0xff03-showme")

	b, err := json.Marshal(ty)
	t.NoError(err)

	var unmarshaled Type
	t.NoError(json.Unmarshal(b, &unmarshaled))
}
