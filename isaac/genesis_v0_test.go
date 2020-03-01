package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testGenesisBlockV0 struct {
	baseTestStateHandler
	localstate *Localstate
}

func (t *testGenesisBlockV0) SetupTest() {
	t.baseTestStateHandler.SetupTest()
	baseLocalstate := t.baseTestStateHandler.localstate

	localstate, err := NewLocalstate(
		NewMemStorage(baseLocalstate.Storage().Encoders(), baseLocalstate.Storage().Encoder()),
		baseLocalstate.Node(),
		TestNetworkID,
	)
	t.NoError(err)
	t.localstate = localstate
}

func (t *testGenesisBlockV0) TestNewGenesisBlock() {
	gg, err := NewGenesisBlockV0Generator(t.localstate, nil)
	t.NoError(err)

	block, err := gg.Generate()
	t.NoError(err)

	t.Equal(Height(0), block.Height())
	t.Equal(Round(0), block.Round())

	pr, err := t.localstate.Storage().Seal(block.Proposal())
	t.NoError(err)
	t.NotNil(pr)
}

func TestGenesisBlockV0(t *testing.T) {
	suite.Run(t, new(testGenesisBlockV0))
}
