package basicstates

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/stretchr/testify/suite"
)

type testStateSwitchContext struct {
	suite.Suite
}

func (t *testStateSwitchContext) TestNew() {
	sctx := NewStateSwitchContext(base.StateBooting, base.StateJoining)
	t.NoError(sctx.IsValid(nil))
}

func (t *testStateSwitchContext) TestUnknownState() {
	t.Run("unknown from state", func() {
		sctx := NewStateSwitchContext(base.StateEmpty, base.StateJoining)
		err := sctx.IsValid(nil)
		t.NotNil(err)
		t.Contains(err.Error(), "invalid state found")
	})

	t.Run("unknown to state", func() {
		sctx := NewStateSwitchContext(base.StateJoining, base.StateEmpty)
		err := sctx.IsValid(nil)
		t.NotNil(err)
		t.Contains(err.Error(), "invalid state found")
	})

	t.Run("unknown from state, but force", func() {
		sctx := NewStateSwitchContext(base.StateEmpty, base.StateJoining).allowEmpty(true)
		t.NoError(sctx.IsValid(nil))
	})
}

func TestStateSwitchContext(t *testing.T) {
	suite.Run(t, new(testStateSwitchContext))
}
