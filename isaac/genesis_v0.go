package isaac

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type GenesisBlockV0Generator struct {
	*logging.Logging
	local     *node.Local
	database  storage.Database
	blockData blockdata.BlockData
	policy    *LocalPolicy
	nodepool  *network.Nodepool
	ballotbox *Ballotbox
	ops       []operation.Operation
	suffrage  base.Suffrage
}

func NewGenesisBlockV0Generator(
	local *node.Local,
	st storage.Database,
	blockData blockdata.BlockData,
	policy *LocalPolicy,
	ops []operation.Operation,
) (*GenesisBlockV0Generator, error) {
	threshold, _ := base.NewThreshold(1, 100)

	suffrage := base.NewFixedSuffrage(local.Address(), nil)
	if err := suffrage.Initialize(); err != nil {
		return nil, err
	}

	nodepool := network.NewNodepool(local, nil)

	return &GenesisBlockV0Generator{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "genesis-block-generator")
		}),
		local:     local,
		database:  st,
		blockData: blockData,
		policy:    policy,
		nodepool:  nodepool,
		ballotbox: NewBallotbox(
			func() []base.Address {
				return []base.Address{local.Address()}
			},
			func() base.Threshold {
				return threshold
			},
		),
		ops:      ops,
		suffrage: suffrage,
	}, nil
}

func (gg *GenesisBlockV0Generator) Generate() (block.Block, error) {
	if err := gg.generatePreviousBlock(); err != nil {
		return nil, err
	}

	ivp, err := gg.generateINITVoteproof()
	if err != nil {
		return nil, err
	}

	seals, err := gg.generateOperationSeal()
	if err != nil {
		return nil, err
	}

	proposal, err := gg.generateProposal(seals, ivp)
	if err != nil {
		return nil, err
	}

	pps := prprocessor.NewProcessors(
		NewDefaultProcessorNewFunc(
			gg.database,
			gg.blockData,
			gg.nodepool,
			gg.suffrage,
			nil,
		),
		nil,
	)
	if err := pps.Initialize(); err != nil {
		return nil, err
	} else if err := pps.Start(); err != nil {
		return nil, err
	} else {
		defer func() {
			_ = pps.Stop()
		}()
	}

	_ = pps.SetLogging(gg.Logging)

	if result := <-pps.NewProposal(context.Background(), proposal, ivp); result.Err != nil {
		return nil, result.Err
	} else if avp, err := gg.generateACCEPTVoteproof(result.Block, ivp); err != nil {
		return nil, err
	} else if result := <-pps.Save(context.Background(), proposal.Hash(), avp); result.Err != nil {
		return nil, result.Err
	} else {
		return pps.Current().Block(), nil
	}
}

func (gg *GenesisBlockV0Generator) generateOperationSeal() ([]operation.Seal, error) {
	if len(gg.ops) < 1 {
		return nil, nil
	}

	var seals []operation.Seal
	if sl, err := operation.NewBaseSeal(
		gg.local.Privatekey(),
		gg.ops,
		gg.policy.NetworkID(),
	); err != nil {
		return nil, err
	} else if err := gg.database.NewSeals([]seal.Seal{sl}); err != nil {
		return nil, err
	} else {
		seals = append(seals, sl)
	}

	return seals, nil
}

func (gg *GenesisBlockV0Generator) generatePreviousBlock() error {
	// NOTE the privatekey of local node is melted into genesis previous block;
	// it means, genesis block contains who creates it.
	sig, err := gg.local.Privatekey().Sign(gg.policy.NetworkID())
	if err != nil {
		return err
	}
	genesisHash := valuehash.NewBytes(sig.Bytes())

	blk, err := block.NewBlockV0(
		block.NewSuffrageInfoV0(
			gg.local.Address(),
			[]base.Node{gg.local},
		),
		base.PreGenesisHeight,
		base.Round(0),
		genesisHash,
		genesisHash,
		nil,
		nil,
		localtime.UTCNow(),
	)
	if err != nil {
		return err
	}

	bs, err := gg.database.NewSession(blk)
	if err != nil {
		return err
	}

	defer func() {
		_ = bs.Close()
	}()

	var bd block.BlockDataMap
	if session, err := gg.blockData.NewSession(blk.Height()); err != nil {
		return err
	} else if err := session.SetBlock(blk); err != nil {
		return err
	} else if i, err := gg.blockData.SaveSession(session); err != nil {
		return err
	} else {
		bd = i
	}

	return bs.Commit(context.Background(), bd)
}

func (gg *GenesisBlockV0Generator) generateProposal(
	seals []operation.Seal,
	voteproof base.Voteproof,
) (ballot.Proposal, error) {
	sealHashes := make([]valuehash.Hash, len(seals))
	for i := range seals {
		sl := seals[i]
		sealHashes[i] = sl.Hash()
	}

	var proposal ballot.Proposal
	pr := ballot.NewProposalV0(
		gg.local.Address(),
		base.Height(0),
		base.Round(0),
		sealHashes,
		voteproof,
	)
	if err := pr.Sign(gg.local.Privatekey(), gg.policy.NetworkID()); err != nil {
		return nil, err
	} else if err := gg.database.NewProposal(pr); err != nil {
		return nil, err
	} else {
		proposal = pr
	}

	return proposal, nil
}

func (gg *GenesisBlockV0Generator) generateINITVoteproof() (base.Voteproof, error) {
	var ib ballot.INITV0
	if b, err := NewINITBallotV0Round0(gg.local, gg.database); err != nil {
		return nil, err
	} else if err := b.Sign(gg.local.Privatekey(), gg.policy.NetworkID()); err != nil {
		return nil, err
	} else {
		ib = b
	}

	voteproof, err := gg.ballotbox.Vote(ib)
	if err != nil {
		return nil, err
	} else if !voteproof.IsFinished() {
		return nil, xerrors.Errorf("something wrong, INITVoteproof should be finished, but not")
	}

	return voteproof, nil
}

func (gg *GenesisBlockV0Generator) generateACCEPTVoteproof(newBlock block.Block, ivp base.Voteproof) (
	base.Voteproof, error,
) {
	ab := NewACCEPTBallotV0(gg.local.Address(), newBlock, ivp)
	if err := ab.Sign(gg.local.Privatekey(), gg.policy.NetworkID()); err != nil {
		return nil, err
	}

	voteproof, err := gg.ballotbox.Vote(ab)
	if err != nil {
		return nil, err
	} else if !voteproof.IsFinished() {
		return nil, xerrors.Errorf("something wrong, ACCEPTVoteproof should be finished, but not")
	}

	return voteproof, nil
}
