package basicstates

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/stretchr/testify/suite"
)

type testStateBooting struct {
	baseTestState
}

func (t *testStateBooting) TestWithBlock() {
	st := NewBootingState(t.local.Node(), t.local.Database(), t.local.Blockdata(), t.local.Policy(), t.Suffrage(t.local))
	defer st.Exit(NewStateSwitchContext(base.StateBooting, base.StateStopped))

	st.SetLastVoteproofFuncs(
		func() base.Voteproof { return nil },
		func() base.Voteproof { return nil },
		func(base.Voteproof) bool { return true },
	)
	f, err := st.Enter(NewStateSwitchContext(base.StateStopped, base.StateBooting))
	t.NoError(err)
	err = f()

	var sctx StateSwitchContext
	t.True(errors.As(err, &sctx))

	t.Equal(base.StateBooting, sctx.FromState())
	t.Equal(base.StateJoining, sctx.ToState())
}

func (t *testStateBooting) TestNoneSuffrageNode() {
	st := NewBootingState(t.local.Node(), t.local.Database(), t.local.Blockdata(), t.local.Policy(), t.Suffrage(t.remote))
	defer st.Exit(NewStateSwitchContext(base.StateBooting, base.StateStopped))

	st.SetLastVoteproofFuncs(
		func() base.Voteproof { return nil },
		func() base.Voteproof { return nil },
		func(base.Voteproof) bool { return true },
	)
	f, err := st.Enter(NewStateSwitchContext(base.StateStopped, base.StateBooting))
	t.NoError(err)
	err = f()

	var sctx StateSwitchContext
	t.True(errors.As(err, &sctx))

	t.Equal(base.StateBooting, sctx.FromState())
	t.Equal(base.StateSyncing, sctx.ToState())
}

func (t *testStateBooting) TestWithEmptyBlockWithChannels() {
	t.NoError(blockdata.Clean(t.local.Database(), t.local.Blockdata(), false))

	st := NewBootingState(t.local.Node(), t.local.Database(), t.local.Blockdata(), t.local.Policy(), t.Suffrage(t.local, t.remote))
	defer t.exitState(st, NewStateSwitchContext(base.StateBooting, base.StateStopped))

	st.syncableChannelsFunc = func() map[string]network.Channel {
		return map[string]network.Channel{
			"showme": t.remote.Channel(),
		}
	}

	f, err := st.Enter(NewStateSwitchContext(base.StateStopped, base.StateBooting))
	t.NoError(err)
	err = f()

	var sctx StateSwitchContext
	t.True(errors.As(err, &sctx))

	t.Equal(base.StateBooting, sctx.FromState())
	t.Equal(base.StateSyncing, sctx.ToState())
}

func (t *testStateBooting) TestWithEmptyBlockWithoutChannels() {
	t.NoError(blockdata.Clean(t.local.Database(), t.local.Blockdata(), false))

	st := NewBootingState(t.local.Node(), t.local.Database(), t.local.Blockdata(), t.local.Policy(), t.Suffrage(t.local))
	defer t.exitState(st, NewStateSwitchContext(base.StateBooting, base.StateStopped))

	st.syncableChannelsFunc = func() map[string]network.Channel {
		return nil
	}

	_, err := st.Enter(NewStateSwitchContext(base.StateStopped, base.StateBooting))
	t.Contains(err.Error(), "empty blocks, but no channels for syncing; can not sync")
}

func TestStateBooting(t *testing.T) {
	suite.Run(t, new(testStateBooting))
}
