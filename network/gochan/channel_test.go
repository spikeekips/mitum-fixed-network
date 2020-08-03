package channetwork

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/util/valuehash"
)

type testNetworkChanChannel struct {
	suite.Suite

	pk key.Privatekey
}

func (t *testNetworkChanChannel) SetupSuite() {
	t.pk, _ = key.NewBTCPrivatekey()
}

func (t *testNetworkChanChannel) TestSendReceive() {
	gs := NewChannel(0)

	sl := seal.NewDummySeal(t.pk)
	go func() {
		_ = gs.SendSeal(sl)
	}()

	rsl := <-gs.ReceiveSeal()

	t.True(sl.Hash().Equal(rsl.Hash()))
}

func (t *testNetworkChanChannel) TestGetSeal() {
	gs := NewChannel(0)

	sl := seal.NewDummySeal(t.pk)

	gs.SetGetSealHandler(func([]valuehash.Hash) ([]seal.Seal, error) {
		return []seal.Seal{sl}, nil
	})

	gsls, err := gs.Seals([]valuehash.Hash{sl.Hash()})
	t.NoError(err)
	t.Equal(1, len(gsls))

	t.True(sl.Hash().Equal(gsls[0].Hash()))
}

func (t *testNetworkChanChannel) TestManifests() {
	gs := NewChannel(0)

	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(9), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	gs.SetGetManifestsHandler(func(heights []base.Height) ([]block.Manifest, error) {
		var blocks []block.Manifest
		for _, h := range heights {
			if h != blk.Height() {
				continue
			}

			blocks = append(blocks, blk.Manifest())
		}

		return blocks, nil
	})

	{
		blocks, err := gs.Manifests([]base.Height{blk.Height()})
		t.NoError(err)
		t.Equal(1, len(blocks))

		for _, b := range blocks {
			_, ok := b.(block.Block)
			t.False(ok)
		}

		t.True(blk.Hash().Equal(blocks[0].Hash()))
	}

	{ // with unknown height
		blocks, err := gs.Manifests([]base.Height{blk.Height(), blk.Height() + 1})
		t.NoError(err)
		t.Equal(1, len(blocks))

		for _, b := range blocks {
			_, ok := b.(block.Block)
			t.False(ok)
		}

		t.True(blk.Hash().Equal(blocks[0].Hash()))
	}
}

func (t *testNetworkChanChannel) TestBlocks() {
	gs := NewChannel(0)

	blk, err := block.NewTestBlockV0(base.Height(33), base.Round(9), valuehash.RandomSHA256(), valuehash.RandomSHA256())
	t.NoError(err)

	gs.SetGetBlocksHandler(func(heights []base.Height) ([]block.Block, error) {
		var blocks []block.Block
		for _, h := range heights {
			if h != blk.Height() {
				continue
			}

			blocks = append(blocks, blk)
		}

		return blocks, nil
	})

	{
		blocks, err := gs.Blocks([]base.Height{blk.Height()})
		t.NoError(err)
		t.Equal(1, len(blocks))

		t.True(blk.Hash().Equal(blocks[0].Hash()))
	}

	{ // with unknown height
		blocks, err := gs.Blocks([]base.Height{blk.Height(), blk.Height() + 1})
		t.NoError(err)
		t.Equal(1, len(blocks))

		t.True(blk.Hash().Equal(blocks[0].Hash()))
	}
}

func TestNetworkChanChannel(t *testing.T) {
	suite.Run(t, new(testNetworkChanChannel))
}
