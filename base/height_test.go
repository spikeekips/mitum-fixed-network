package base

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/stretchr/testify/suite"
)

type testHeight struct {
	suite.Suite
}

func (t *testHeight) TestNew() {
	h10 := Height(10)
	t.Equal(int64(10), int64(h10))
}

func (t *testHeight) TestInt64() {
	h10 := Height(10)
	t.Equal(int64(10), h10.Int64())
}

func (t *testHeight) TestInvalid() {
	h10 := Height(10)
	t.NoError(h10.IsValid(nil))

	hu1 := Height(NilHeight)
	t.True(errors.Is(isvalid.InvalidError, hu1.IsValid(nil)))
}

func TestHeight(t *testing.T) {
	suite.Run(t, new(testHeight))
}
