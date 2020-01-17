package hint

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
)

type testHintVersion struct {
	suite.Suite
}

func (t *testHintVersion) TestNew() {
	ty := Type([2]byte{0xff, 0xf0})
	v := Version("0.1")

	h, err := NewHint(ty, v)
	t.NoError(err)
	t.Equal(ty, h.Type())
	t.Equal(v, h.Version())
}

func (t *testHintVersion) TestInvalidVersion() {
	_, err := NewHint(
		Type([2]byte{0xff, 0xf0}),
		Version("vv0.1"),
	)
	t.True(xerrors.Is(err, InvalidVersionError))
}

func TestHintVersion(t *testing.T) {
	suite.Run(t, new(testHintVersion))
}

type testHint struct {
	suite.Suite
}

func (t *testHint) TestNew() {
	ty := Type([2]byte{0xff, 0xf0})
	v := Version("0.1")

	hint, err := NewHint(ty, v)
	t.NoError(err)

	t.Equal(ty, hint.Type())
	t.Equal(v, hint.Version())
}

func (t *testHint) TestWrongSizeVersion() {
	ty := Type([2]byte{0xff, 0xf0})
	v := Version("0.1-" + strings.Repeat("k", MaxVersionSize-3))

	_, err := NewHint(ty, v)
	t.True(xerrors.Is(err, InvalidVersionError))
	t.Contains(err.Error(), "oversized version")
}

func (t *testHint) TestInvalidType() {
	ty := NullType
	v := Version("0.1")

	_, err := NewHint(ty, v)
	t.True(xerrors.Is(err, InvalidTypeError))
	t.Contains(err.Error(), "empty")
}

func (t *testHint) TestBytes() {
	ty := Type([2]byte{0xff, 0xf0})
	v := Version("0.1")

	hint, err := NewHint(ty, v)
	t.NoError(err)

	t.True(2+MaxVersionSize >= len(hint.Bytes()))

	nh, err := NewHintFromBytes(hint.Bytes())
	t.NoError(err)

	t.Equal(hint.Type(), nh.Type())
	t.Equal(hint.Version(), nh.Version())
}

func (t *testHint) TestCompatible() {
	cases := []struct {
		name string
		t0   [2]byte
		v0   string
		t1   [2]byte
		v1   string
		err  error
	}{
		{
			name: "same type and version",
			t0:   [2]byte{0xff, 0xf0},
			v0:   "0.1.0",
			t1:   [2]byte{0xff, 0xf0},
			v1:   "0.1.0",
		},
		{
			name: "lower patch version",
			t0:   [2]byte{0xff, 0xf0},
			v0:   "0.1.1",
			t1:   [2]byte{0xff, 0xf0},
			v1:   "0.1.0",
		},
		{
			name: "greater patch version",
			t0:   [2]byte{0xff, 0xf0},
			v0:   "0.1.0",
			t1:   [2]byte{0xff, 0xf0},
			v1:   "0.1.1",
			err:  VersionNotCompatibleError,
		},
		{
			name: "lower minor version",
			t0:   [2]byte{0xff, 0xf0},
			v0:   "0.1.0",
			t1:   [2]byte{0xff, 0xf0},
			v1:   "0.0.9",
		},
		{
			name: "greater major version",
			t0:   [2]byte{0xff, 0xf0},
			v0:   "0.1.0",
			t1:   [2]byte{0xff, 0xf0},
			v1:   "1.0.9",
			err:  VersionNotCompatibleError,
		},
		{
			name: "different type",
			t0:   [2]byte{0xff, 0xf0},
			v0:   "0.1.0",
			t1:   [2]byte{0xff, 0xf1},
			v1:   "0.0.9",
			err:  TypeDoesNotMatchError,
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.Run(
			c.name,
			func() {
				target, _ := NewHint(Type(c.t0), Version(c.v0))
				check, _ := NewHint(Type(c.t1), Version(c.v1))

				err := target.IsCompatible(check)
				if c.err != nil {
					t.True(xerrors.Is(err, c.err), "%d: %v; %v != %v", i, c.name, c.err, err)
				} else if err != nil {
					t.NoError(err, "%d: %v; %v != %v", i, c.name, c.err, err)
				}
			},
		)
	}
}

func TestHint(t *testing.T) {
	suite.Run(t, new(testHint))
}
