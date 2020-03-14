package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type testStateDurationValue struct {
	suite.Suite
}

func (t *testStateDurationValue) TestNew() {
	v := time.Second * 100
	dv, err := NewDurationValue(v)
	t.NoError(err)

	t.Implements((*Value)(nil), dv)

	t.Equal(v, dv.v)
	t.Equal(v, dv.Interface())
	t.NotNil(dv.Hash())
	t.NotNil(dv.Bytes())
}

func TestStateDurationValue(t *testing.T) {
	suite.Run(t, new(testStateDurationValue))
}
