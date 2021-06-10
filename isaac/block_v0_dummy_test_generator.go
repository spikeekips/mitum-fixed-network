// +build test

package isaac

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/prprocessor"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/storage/blockdata"
	"github.com/spikeekips/mitum/storage/blockdata/localfs"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/spikeekips/mitum/util/valuehash"
)

type DummyBlocksV0Generator struct {
	*logging.Logging
	genesisNode *Local
	locals      []*Local
	lastHeight  base.Height
	suffrage    base.Suffrage
	networkID   []byte
	allNodes    map[base.Address]*Local
	ballotboxes map[base.Address]*Ballotbox
	ppss        map[base.Address]*prprocessor.Processors
}

func NewDummyBlocksV0Generator(
	genesisNode *Local, lastHeight base.Height, suffrage base.Suffrage, locals []*Local,
) (*DummyBlocksV0Generator, error) {
	if lastHeight <= base.NilHeight {
		return nil, xerrors.Errorf("last height must not be nil height, %v", base.NilHeight)
	}

	allNodes := map[base.Address]*Local{}
	ballotboxes := map[base.Address]*Ballotbox{}
	pms := map[base.Address]*prprocessor.Processors{}

	threshold, _ := base.NewThreshold(uint(len(locals)), 67)
	for _, l := range locals {
		allNodes[l.Node().Address()] = l
		ballotboxes[l.Node().Address()] = NewBallotbox(
			suffrage.Nodes,
			func() base.Threshold {
				return threshold
			},
		)
		pps := prprocessor.NewProcessors(
			NewDefaultProcessorNewFunc(l.Database(), l.BlockData(), l.Nodes(), suffrage, nil),
			nil,
		)
		if err := pps.Initialize(); err != nil {
			return nil, err
		} else if err := pps.Start(); err != nil {
			return nil, err
		}

		pms[l.Node().Address()] = pps
	}

	return &DummyBlocksV0Generator{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "dummy-block-generator")
		}),
		genesisNode: genesisNode,
		locals:      locals,
		lastHeight:  lastHeight,
		suffrage:    suffrage,
		networkID:   genesisNode.Policy().NetworkID(),
		allNodes:    allNodes,
		ballotboxes: ballotboxes,
		ppss:        pms,
	}, nil
}

func (bg *DummyBlocksV0Generator) Close() error {
	for _, pps := range bg.ppss {
		if err := pps.Stop(); err != nil {
			panic(err) // DummyBlocksV0Generator used only for testing
		}
	}

	return nil
}

func (bg *DummyBlocksV0Generator) findLastHeight() (base.Height, error) {
	switch l, found, err := bg.genesisNode.Database().LastManifest(); {
	case err != nil:
		return base.NilHeight, err
	case !found:
		return base.NilHeight, nil
	default:
		switch err := l.IsValid(bg.networkID); {
		case err != nil:
			return base.NilHeight, err
		default:
			return l.Height(), nil
		}
	}
}

func (bg *DummyBlocksV0Generator) Generate(ignoreExists bool) error {
	defer func() {
		_ = bg.Close()
	}()

	if ignoreExists {
		for _, n := range bg.allNodes {
			if err := blockdata.Clean(n.Database(), n.BlockData(), false); err != nil {
				return err
			}
		}
	}

	lastHeight := base.NilHeight
	if !ignoreExists {
		switch h, err := bg.findLastHeight(); {
		case err != nil:
			return err
		case h >= bg.lastHeight:
			return nil
		default:
			lastHeight = h
		}
	}

	if lastHeight == base.NilHeight {
		if genesis, err := NewGenesisBlockV0Generator(
			bg.genesisNode.Node(),
			bg.genesisNode.Database(),
			bg.genesisNode.BlockData(),
			bg.genesisNode.Policy(),
			nil,
		); err != nil {
			return err
		} else {
			_ = genesis.SetLogger(bg.Log())

			if _, err := genesis.Generate(); err != nil {
				return err
			} else if err := bg.syncBlocks(bg.genesisNode); err != nil {
				return err
			}
		}
	}

	if bg.lastHeight == base.PreGenesisHeight+1 {
		return nil
	}

end:
	for {
		if err := bg.createNextBlock(); err != nil {
			return err
		}

		switch l, found, err := bg.genesisNode.Database().LastManifest(); {
		case !found:
			return util.NotFoundError.Errorf("last manifest not found")
		case err != nil:
			return err
		case l.Height() == bg.lastHeight:
			break end
		}
	}

	return nil
}

func (bg *DummyBlocksV0Generator) syncBlocks(from *Local) error {
	var blocks []block.Block
	height := base.PreGenesisHeight

	fbs := from.BlockData().(*localfs.BlockData)

end:
	for {
		switch _, blk, err := localfs.LoadBlock(fbs, height); {
		case err != nil:
			if xerrors.Is(err, util.NotFoundError) {
				break end
			}

			return err
		default:
			blocks = append(blocks, blk)
		}

		height++
	}

	if len(blocks) < 1 {
		return xerrors.Errorf("empty blocks for syncing blocks")
	}

	for _, blk := range blocks {
		for _, l := range bg.allNodes {
			if l.Node().Address().Equal(from.Node().Address()) {
				continue
			}

			if err := bg.storeBlock(l, blk); err != nil {
				return err
			}
		}
	}

	return bg.syncSeals(from)
}

func (bg *DummyBlocksV0Generator) storeBlock(l *Local, blk block.Block) error {
	var bs storage.DatabaseSession
	if st, err := l.Database().NewSession(blk); err != nil {
		return err
	} else {
		bs = st
	}

	defer func() {
		_ = bs.Close()
	}()

	var session blockdata.Session
	if i, err := l.BlockData().NewSession(blk.Height()); err != nil {
		return err
	} else {
		session = i
	}

	var bd block.BlockDataMap
	if err := session.SetBlock(blk); err != nil {
		return err
	} else if i, err := l.BlockData().SaveSession(session); err != nil {
		return err
	} else {
		bd = i
	}

	if err := bs.SetBlock(context.Background(), blk); err != nil {
		return err
	} else if err := bs.Commit(context.Background(), bd); err != nil {
		return err
	}

	return nil
}

func (bg *DummyBlocksV0Generator) syncSeals(from *Local) error {
	var seals []seal.Seal
	if err := from.Database().Seals(
		func(_ valuehash.Hash, sl seal.Seal) (bool, error) {
			seals = append(seals, sl)
			return true, nil
		},
		true,
		true,
	); err != nil {
		return err
	}

	for _, l := range bg.allNodes {
		if l.Node().Address().Equal(from.Node().Address()) {
			continue
		}

		if err := l.Database().NewSeals(seals); err != nil {
			return err
		}
	}

	var proposals []ballot.Proposal
	if err := from.Database().Proposals(
		func(proposal ballot.Proposal) (bool, error) {
			proposals = append(proposals, proposal)
			return true, nil
		},
		true,
	); err != nil {
		return err
	}

	for _, l := range bg.allNodes {
		if l.Node().Address().Equal(from.Node().Address()) {
			continue
		}

		for _, proposal := range proposals {
			if err := l.Database().NewProposal(proposal); err != nil {
				if xerrors.Is(err, util.DuplicatedError) {
					continue
				}

				return err
			}
		}
	}

	return nil
}

func (bg *DummyBlocksV0Generator) createNextBlock() error {
	var ivm map[base.Address]base.Voteproof
	if v, err := bg.createINITVoteproof(); err != nil {
		return err
	} else {
		ivm = v
	}

	var proposal ballot.Proposal
	if pr, err := bg.createProposal(ivm[bg.genesisNode.Node().Address()]); err != nil {
		return err
	} else {
		proposal = pr
	}

	var avm map[base.Address]base.Voteproof
	if v, err := bg.createACCEPTVoteproof(proposal, ivm); err != nil {
		return err
	} else {
		avm = v
	}

	for _, l := range bg.allNodes {
		var vp base.Voteproof
		if v, found := avm[l.Node().Address()]; !found {
			return xerrors.Errorf("failed to find voteproofs for all nodes")
		} else {
			vp = v
		}

		if err := bg.finish(l, vp); err != nil {
			return err
		}
	}

	return nil
}

func (bg *DummyBlocksV0Generator) finish(l *Local, vp base.Voteproof) error {
	proposal := vp.Majority().(ballot.ACCEPTFact).Proposal()

	pps := bg.ppss[l.Node().Address()]
	if result := <-pps.Save(context.Background(), proposal, vp); result.Err != nil {
		return result.Err
	}

	return nil
}

func (bg *DummyBlocksV0Generator) createINITVoteproof() (map[base.Address]base.Voteproof, error) {
	var ballots []ballot.INIT
	for _, l := range bg.allNodes {
		if ib, err := bg.createINITBallot(l); err != nil {
			return nil, err
		} else {
			ballots = append(ballots, ib)
		}
	}

	vm := map[base.Address]base.Voteproof{}
	for _, l := range bg.allNodes {
		for _, blt := range ballots {
			if voteproof, err := bg.ballotboxes[l.Node().Address()].Vote(blt); err != nil {
				return nil, err
			} else if voteproof.IsFinished() && !voteproof.IsClosed() {
				vm[l.Node().Address()] = voteproof
			}
		}
	}

	if len(vm) != len(bg.allNodes) {
		return nil, xerrors.Errorf("failed to create INIT Voteproof")
	}

	return vm, nil
}

func (bg *DummyBlocksV0Generator) createINITBallot(local *Local) (ballot.INIT, error) {
	var baseBallot ballot.INITV0
	if b, err := NewINITBallotV0Round0(local.Node(), local.Database()); err != nil {
		return nil, err
	} else if err := SignSeal(&b, local); err != nil {
		return nil, err
	} else {
		baseBallot = b
	}

	return baseBallot, nil
}

func (bg *DummyBlocksV0Generator) createProposal(voteproof base.Voteproof) (ballot.Proposal, error) {
	var proposer *Local
	if acting, err := bg.suffrage.Acting(voteproof.Height(), voteproof.Round()); err != nil {
		return nil, err
	} else if acting.Proposer() == nil {
		return nil, xerrors.Errorf("empty proposer")
	} else {
		proposer = bg.allNodes[acting.Proposer()]
	}

	pr := ballot.NewProposalV0(
		proposer.Node().Address(),
		voteproof.Height(),
		voteproof.Round(),
		nil,
		voteproof,
	)
	if err := SignSeal(&pr, proposer); err != nil {
		return nil, err
	}

	for _, l := range bg.allNodes {
		if err := l.Database().NewProposal(pr); err != nil {
			return nil, err
		}
	}

	return pr, nil
}

func (bg *DummyBlocksV0Generator) createACCEPTVoteproof(proposal ballot.Proposal, ivm map[base.Address]base.Voteproof) (
	map[base.Address]base.Voteproof, error,
) {
	var ballots []ballot.ACCEPT
	for _, l := range bg.allNodes {
		var newBlock block.Block

		ivp := ivm[l.Node().Address()]
		pps := bg.ppss[l.Node().Address()]
		if result := <-pps.NewProposal(context.Background(), proposal, ivp); result.Err != nil {
			return nil, result.Err
		} else {
			newBlock = result.Block
		}

		ab := NewACCEPTBallotV0(l.Node().Address(), newBlock, ivp)
		if err := SignSeal(&ab, l); err != nil {
			return nil, err
		} else {
			ballots = append(ballots, ab)
		}
	}

	vm := map[base.Address]base.Voteproof{}
	for _, l := range bg.allNodes {
		for _, blt := range ballots {
			if voteproof, err := bg.ballotboxes[l.Node().Address()].Vote(blt); err != nil {
				return nil, err
			} else if voteproof.IsFinished() && !voteproof.IsClosed() {
				vm[l.Node().Address()] = voteproof
			}
		}
	}

	if len(vm) != len(bg.allNodes) {
		return nil, xerrors.Errorf("failed to create voteproofs for all nodes")
	}

	return vm, nil
}
