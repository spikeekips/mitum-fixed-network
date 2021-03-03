package basicstates

import (
	"testing"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/storage"
	"github.com/stretchr/testify/suite"
)

type testStateBooting struct {
	baseTestState
	local  *isaac.Local
	remote *isaac.Local
}

func (t *testStateBooting) SetupTest() {
	ls := t.Locals(2)
	t.local, t.remote = ls[0], ls[1]
}

func (t *testStateBooting) TestWithBlock() {
	st := NewBootingState(t.local.Storage(), t.local.BlockFS(), t.local.Policy(), t.Suffrage(t.local))
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

func (t *testStateBooting) TestWithEmptyBlockWithSuffrageNodes() {
	t.NoError(storage.Clean(t.local.Storage(), t.local.BlockFS(), false))

	st := NewBootingState(t.local.Storage(), t.local.BlockFS(), t.local.Policy(), t.Suffrage(t.local, t.remote))
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
	t.NoError(storage.Clean(t.local.Storage(), t.local.BlockFS(), false))

	st := NewBootingState(t.local.Storage(), t.local.BlockFS(), t.local.Policy(), t.Suffrage(t.local))
	defer t.exitState(st, NewStateSwitchContext(base.StateBooting, base.StateStopped))

	_, err := st.Enter(NewStateSwitchContext(base.StateStopped, base.StateBooting))
	t.Contains(err.Error(), "empty blocks, but no other nodes; can not sync")
}

func TestStateBooting(t *testing.T) {
	suite.Run(t, new(testStateBooting))
}
