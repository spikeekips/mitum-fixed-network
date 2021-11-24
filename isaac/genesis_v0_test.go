package isaac

import (
	"testing"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/stretchr/testify/suite"
)

type testGenesisBlockV0 struct {
	BaseTest
	local *Local
}

func (t *testGenesisBlockV0) SetupTest() {
	t.BaseTest.SetupTest()

	t.local = t.Locals(1)[0]
	t.local.Database().Clean()
}

func (t *testGenesisBlockV0) TestNewGenesisBlock() {
	op, err := NewKVOperation(
		t.local.Node().Privatekey(),
		[]byte("this-is-token"),
		"showme",
		[]byte("findme"),
		nil,
	)
	t.NoError(err)

	gg, err := NewGenesisBlockV0Generator(t.local.Node(), t.local.Database(), t.local.BlockData(), t.local.Policy(), []operation.Operation{op})
	t.NoError(err)

	blk, err := gg.Generate()
	t.NoError(err)

	t.Equal(base.GenesisHeight, blk.Height())
	t.Equal(base.Round(0), blk.Round())

	var found bool
	t.NoError(t.local.Database().Proposals(func(proposal base.Proposal) (bool, error) {
		if proposal.Fact().Hash().Equal(blk.Proposal()) {
			found = true

			return false, nil
		}

		return true, nil
	}, false))

	t.True(found)

	st, found, err := t.local.Database().State(op.Key())
	t.NoError(err)
	t.True(found)

	t.Equal(st.Key(), op.Key())
}

func TestGenesisBlockV0(t *testing.T) {
	suite.Run(t, new(testGenesisBlockV0))
}
