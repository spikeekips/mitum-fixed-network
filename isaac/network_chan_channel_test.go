package isaac

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/seal"
	"github.com/spikeekips/mitum/valuehash"
)

type testNetworkChanChannel struct {
	suite.Suite

	pk key.BTCPrivatekey
}

func (t *testNetworkChanChannel) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testNetworkChanChannel) TestSendReceive() {
	gs := NewNetworkChanChannel(0)

	sl := seal.NewDummySeal(t.pk)
	go func() {
		t.NoError(gs.SendSeal(sl))
	}()

	rsl := <-gs.ReceiveSeal()

	t.True(sl.Hash().Equal(rsl.Hash()))
}

func (t *testNetworkChanChannel) TestGetSeal() {
	gs := NewNetworkChanChannel(0)

	sl := seal.NewDummySeal(t.pk)

	gs.SetGetSealHandler(func([]valuehash.Hash) ([]seal.Seal, error) {
		return []seal.Seal{sl}, nil
	})

	gsls, err := gs.Seals([]valuehash.Hash{sl.Hash()})
	t.NoError(err)
	t.Equal(1, len(gsls))

	t.True(sl.Hash().Equal(gsls[0].Hash()))
}

func (t *testNetworkChanChannel) TestBlockManifests() {
	gs := NewNetworkChanChannel(0)

	block, err := NewTestBlockV0(Height(33), Round(9), nil, valuehash.RandomSHA256())
	t.NoError(err)

	gs.SetGetBlockManifests(func(heights []Height) ([]BlockManifest, error) {
		var blocks []BlockManifest
		for _, h := range heights {
			if h != block.Height() {
				continue
			}

			blocks = append(blocks, block.Manifest())
		}

		return blocks, nil
	})

	{
		blocks, err := gs.BlockManifests([]Height{block.Height()})
		t.NoError(err)
		t.Equal(1, len(blocks))

		for _, b := range blocks {
			_, ok := b.(Block)
			t.False(ok)
		}

		t.True(block.Hash().Equal(blocks[0].Hash()))
	}

	{ // with unknown height
		blocks, err := gs.BlockManifests([]Height{block.Height(), block.Height() + 1})
		t.NoError(err)
		t.Equal(1, len(blocks))

		for _, b := range blocks {
			_, ok := b.(Block)
			t.False(ok)
		}

		t.True(block.Hash().Equal(blocks[0].Hash()))
	}
}

func (t *testNetworkChanChannel) TestBlocks() {
	gs := NewNetworkChanChannel(0)

	block, err := NewTestBlockV0(Height(33), Round(9), nil, valuehash.RandomSHA256())
	t.NoError(err)

	gs.SetGetBlocks(func(heights []Height) ([]Block, error) {
		var blocks []Block
		for _, h := range heights {
			if h != block.Height() {
				continue
			}

			blocks = append(blocks, block)
		}

		return blocks, nil
	})

	{
		blocks, err := gs.Blocks([]Height{block.Height()})
		t.NoError(err)
		t.Equal(1, len(blocks))

		t.True(block.Hash().Equal(blocks[0].Hash()))
	}

	{ // with unknown height
		blocks, err := gs.Blocks([]Height{block.Height(), block.Height() + 1})
		t.NoError(err)
		t.Equal(1, len(blocks))

		t.True(block.Hash().Equal(blocks[0].Hash()))
	}
}

func TestNetworkChanChannel(t *testing.T) {
	suite.Run(t, new(testNetworkChanChannel))
}
