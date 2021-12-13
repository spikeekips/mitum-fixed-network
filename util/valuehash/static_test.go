package valuehash

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/stretchr/testify/suite"
)

func newL32(b []byte) (L32, error) {
	if len(b) != 32 {
		return L32{}, fmt.Errorf("invalid length for L32")
	}

	var t [32]byte
	copy(t[:], b)

	return L32(t), nil
}

func newL64(b []byte) (L64, error) {
	if len(b) != 64 {
		return L64{}, fmt.Errorf("invalid length for L64")
	}

	var t [64]byte
	copy(t[:], b)

	return L64(t), nil
}

type testStatic struct {
	suite.Suite
}

func (t *testStatic) TestNew32() {
	b := bytes.Repeat([]byte("1"), 32)
	h, err := newL32(b)
	t.NoError(err)

	_, ok := (interface{})(h).(Hash)
	t.True(ok)

	t.NoError(h.IsValid(nil))
	t.True(h.Equal(h))
}

func (t *testStatic) TestNew64() {
	b := bytes.Repeat([]byte("1"), 64)
	h, err := newL64(b)
	t.NoError(err)

	_, ok := (interface{})(h).(Hash)
	t.True(ok)

	t.NoError(h.IsValid(nil))
	t.True(h.Equal(h))
}

func (t *testStatic) TestNew32Long() {
	b := bytes.Repeat([]byte("1"), 33)
	_, err := newL32(b)
	t.Error(err)
	t.Contains(err.Error(), "invalid length")
}

func (t *testStatic) TestEmpty32() {
	h := L32{}
	t.True(h.IsEmpty())
}

func (t *testStatic) TestEmpty64() {
	h := L64{}
	t.True(h.IsEmpty())

	err := h.IsValid(nil)
	t.Error(err)
	t.True(errors.Is(err, isvalid.InvalidError))
}

func (t *testStatic) TestString() {
	b := bytes.Repeat([]byte("1"), 32)
	ha, err := newL32(b)
	t.NoError(err)

	hb, err := newL32(b)
	t.NoError(err)

	t.Equal(ha.String(), hb.String())
	t.Equal(b, hb.Bytes())
}

func (t *testStatic) TestBytes() {
	b := bytes.Repeat([]byte("1"), 32)
	ha, err := newL32(b)
	t.NoError(err)

	hb, err := newL32(b)
	t.NoError(err)

	t.Equal(ha.Bytes(), hb.Bytes())
	t.Equal(b, hb.Bytes())
}

func TestStatic(t *testing.T) {
	suite.Run(t, new(testStatic))
}
