package basicstates

import (
	"testing"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/stretchr/testify/suite"
)

type testStateBooting struct {
	baseTestState
}

func (t *testStateBooting) TestWithBlock() {
	st := NewBootingState(t.local.Node(), t.local.Database(), t.local.BlockData(), t.local.Policy(), t.Suffrage(t.local))
	defer st.Exit(NewStateSwitchContext(base.StateBooting, base.StateStopped))

	st.SetLastVoteproofFuncs(
		func() base.Voteproof { return nil },
		func() base.Voteproof { return nil },
		func(base.Voteproof) {},
	)
	f, err := st.Enter(NewStateSwitchContext(base.StateStopped, base.StateBooting))
	t.NoError(err)
	err = f()

	var sctx StateSwitchContext
	t.True(xerrors.As(err, &sctx))

	t.Equal(base.StateBooting, sctx.FromState())
	t.Equal(base.StateJoining, sctx.ToState())
}

func (t *testStateBooting) TestNoneSuffrageNode() {
	st := NewBootingState(t.local.Node(), t.local.Database(), t.local.BlockData(), t.local.Policy(), t.Suffrage(t.remote))
	defer st.Exit(NewStateSwitchContext(base.StateBooting, base.StateStopped))

	st.SetLastVoteproofFuncs(
		func() base.Voteproof { return nil },
		func() base.Voteproof { return nil },
		func(base.Voteproof) {},
	)
	f, err := st.Enter(NewStateSwitchContext(base.StateStopped, base.StateBooting))
	t.NoError(err)
	err = f()

	var sctx StateSwitchContext
	t.True(xerrors.As(err, &sctx))

	t.Equal(base.StateBooting, sctx.FromState())
	t.Equal(base.StateSyncing, sctx.ToState())
}

func (t *testStateBooting) TestWithEmptyBlockWithSuffrageNodes() {
	t.NoError(blockdata.Clean(t.local.Database(), t.local.BlockData(), false))

	st := NewBootingState(t.local.Node(), t.local.Database(), t.local.BlockData(), t.local.Policy(), t.Suffrage(t.local, t.remote))
	defer t.exitState(st, NewStateSwitchContext(base.StateBooting, base.StateStopped))

	f, err := st.Enter(NewStateSwitchContext(base.StateStopped, base.StateBooting))
	t.NoError(err)
	err = f()

	var sctx StateSwitchContext
	t.True(xerrors.As(err, &sctx))

	t.Equal(base.StateBooting, sctx.FromState())
	t.Equal(base.StateSyncing, sctx.ToState())
}

func (t *testStateBooting) TestWithEmptyBlockWithoutSuffrageNodes() {
	t.NoError(blockdata.Clean(t.local.Database(), t.local.BlockData(), false))

	st := NewBootingState(t.local.Node(), t.local.Database(), t.local.BlockData(), t.local.Policy(), t.Suffrage(t.local))
	defer t.exitState(st, NewStateSwitchContext(base.StateBooting, base.StateStopped))

	_, err := st.Enter(NewStateSwitchContext(base.StateStopped, base.StateBooting))
	t.Contains(err.Error(), "empty blocks, but no other nodes; can not sync")
}

func TestStateBooting(t *testing.T) {
	suite.Run(t, new(testStateBooting))
}
