package isaac

import (
	"context"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/node"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type GenesisBlockV0Generator struct {
	*logging.Logging
	local     node.Local
	database  storage.Database
	blockdata blockdata.Blockdata
	policy    *LocalPolicy
	nodepool  *network.Nodepool
	ballotbox *Ballotbox
	ops       []operation.Operation
	suffrage  base.Suffrage
}

func NewGenesisBlockV0Generator(
	local node.Local,
	db storage.Database,
	bd blockdata.Blockdata,
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
		database:  db,
		blockdata: bd,
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

	pr, err := gg.generateProposal(seals, ivp)
	if err != nil {
		return nil, err
	}

	pps := prprocessor.NewProcessors(
		NewDefaultProcessorNewFunc(
			gg.database,
			gg.blockdata,
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

	if result := <-pps.NewProposal(context.Background(), pr.SignedFact(), ivp); result.Err != nil {
		return nil, result.Err
	} else if avp, err := gg.generateACCEPTVoteproof(result.Block, ivp); err != nil {
		return nil, err
	} else if result := <-pps.Save(context.Background(), pr.Fact().Hash(), avp); result.Err != nil {
		return nil, result.Err
	} else {
		return pps.Current().Block(), nil
	}
}

func (gg *GenesisBlockV0Generator) generateOperationSeal() ([]operation.Seal, error) {
	if len(gg.ops) < 1 {
		return nil, nil
	}

	sl, err := operation.NewBaseSeal(
		gg.local.Privatekey(),
		gg.ops,
		gg.policy.NetworkID(),
	)
	if err != nil {
		return nil, err
	}

	seals := []operation.Seal{sl}
	if err = gg.database.NewOperationSeals(seals); err != nil {
		return nil, err
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

	var bd block.BlockdataMap
	if session, err := gg.blockdata.NewSession(blk.Height()); err != nil {
		return err
	} else if err := session.SetBlock(blk); err != nil {
		return err
	} else if i, err := gg.blockdata.SaveSession(session); err != nil {
		return err
	} else {
		bd = i
	}

	return bs.Commit(context.Background(), bd)
}

func (gg *GenesisBlockV0Generator) generateProposal(
	seals []operation.Seal,
	voteproof base.Voteproof,
) (base.Proposal, error) {
	var ops []valuehash.Hash
	for i := range seals {
		l := seals[i].Operations()
		for j := range l {
			ops = append(ops, l[j].Fact().Hash())
		}
	}

	pr, err := ballot.NewProposal(
		ballot.NewProposalFact(
			base.GenesisHeight,
			base.Round(0),
			gg.local.Address(),
			ops,
		),
		gg.local.Address(),
		voteproof,
		gg.local.Privatekey(), gg.policy.NetworkID(),
	)
	if err != nil {
		return nil, err
	}

	if err := gg.database.NewProposal(pr); err != nil {
		return nil, err
	}

	return pr, nil
}

func (gg *GenesisBlockV0Generator) generateINITVoteproof() (base.Voteproof, error) {
	ib, err := NewINITBallotRound0(gg.local.Address(), gg.database, gg.local.Privatekey(), gg.policy.NetworkID())
	if err != nil {
		return nil, err
	}

	voteproof, err := gg.ballotbox.Vote(ib)
	if err != nil {
		return nil, err
	} else if !voteproof.IsFinished() {
		return nil, errors.Errorf("something wrong, INITVoteproof should be finished, but not")
	}

	return voteproof, nil
}

func (gg *GenesisBlockV0Generator) generateACCEPTVoteproof(
	newBlock block.Block, ivp base.Voteproof,
) (base.Voteproof, error) {
	ab, err := NewACCEPTBallot(gg.local.Address(), newBlock, ivp, gg.local.Privatekey(), gg.policy.NetworkID())
	if err != nil {
		return nil, err
	}

	voteproof, err := gg.ballotbox.Vote(ab)
	if err != nil {
		return nil, err
	} else if !voteproof.IsFinished() {
		return nil, errors.Errorf("something wrong, ACCEPTVoteproof should be finished, but not")
	}

	return voteproof, nil
}
