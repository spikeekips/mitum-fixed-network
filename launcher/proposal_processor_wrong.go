package launcher

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type WrongProposalProcessor struct {
	*isaac.DefaultProposalProcessor
	local  *isaac.Local
	points []BlockPoint
}

func NewWrongProposalProcessor(
	local *isaac.Local,
	suffrage base.Suffrage,
	points []BlockPoint,
) *WrongProposalProcessor {
	wp := &WrongProposalProcessor{
		DefaultProposalProcessor: isaac.NewDefaultProposalProcessor(local, suffrage),
		points:                   points,
		local:                    local,
	}
	wp.Logging = logging.NewLogging(func(c logging.Context) logging.Emitter {
		return c.Str("module", "wrong-proposal-processor")
	})

	return wp
}

func (wp *WrongProposalProcessor) ProcessACCEPT(ph valuehash.Hash, voteproof base.Voteproof) (
	storage.BlockStorage, error,
) {
	bs, err := wp.DefaultProposalProcessor.ProcessACCEPT(ph, voteproof)
	if err != nil {
		return nil, err
	}

	var found bool
	for _, h := range wp.points {
		if h.Height == voteproof.Height() && h.Round == voteproof.Round() {
			found = true
			break
		}
	}

	if !found {
		return bs, nil
	}

	// NOTE make fake block
	orig := bs.Block()
	if blk, err := block.NewBlockV0(
		orig.ConsensusInfo().SuffrageInfo().(block.SuffrageInfoV0),
		orig.Height(),
		orig.Round(),
		orig.Proposal(),
		orig.PreviousBlock(),
		valuehash.RandomSHA256(),
		valuehash.RandomSHA256(), // NOTE set random hash for OperationsHash() and StatesHash()
		localtime.Now(),
	); err != nil {
		panic(err)
	} else {
		newBlock := blk.
			SetINITVoteproof(orig.ConsensusInfo().INITVoteproof()).
			SetACCEPTVoteproof(orig.ConsensusInfo().ACCEPTVoteproof()).
			SetProposal(orig.ConsensusInfo().Proposal())

		newbs, err := wp.local.Storage().OpenBlockStorage(newBlock)
		if err != nil {
			panic(err)
		} else if err := newbs.SetBlock(newBlock); err != nil {
			panic(err)
		}

		bs = newbs

		wp.Log().Debug().
			Dict("block", logging.Dict().
				Hinted("height", voteproof.Height()).
				Hinted("round", voteproof.Round()),
			).
			Hinted("original_block", orig.Hash()).
			Hinted("new_block", newBlock.Hash()).
			Msg("made wrong block")
	}

	return bs, nil
}
