package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/isvalid"
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

	hu1 := Height(-1)
	t.True(xerrors.Is(isvalid.InvalidError, hu1.IsValid(nil)))
}

func TestHeight(t *testing.T) {
	suite.Run(t, new(testHeight))
}
