package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
)

type Localstate struct {
	storage             Storage
	node                *LocalNode
	policy              *LocalPolicy
	nodes               *NodesState
	lastBlock           *util.LockedItem
	lastINITVoteproof   *util.LockedItem
	lastACCEPTVoteproof *util.LockedItem
}

func NewLocalstate(st Storage, node *LocalNode, networkID []byte) (*Localstate, error) {
	// load last states from storage.
	var lastBlock block.Block
	var lastINITVoteproof base.Voteproof
	var lastACCEPTVoteproof base.Voteproof
	if st != nil {
		var err error
		if lastBlock, err = st.LastBlock(); err != nil {
			return nil, err
		} else if lastBlock != nil {
			lastINITVoteproof = lastBlock.INITVoteproof()
			lastACCEPTVoteproof = lastBlock.ACCEPTVoteproof()
		}
	}

	var policy *LocalPolicy
	if p, err := NewLocalPolicy(st, networkID); err != nil {
		return nil, err
	} else {
		policy = p
	}

	return &Localstate{
		storage:             st,
		node:                node,
		policy:              policy,
		nodes:               NewNodesState(node, nil),
		lastBlock:           util.NewLockedItem(lastBlock),
		lastINITVoteproof:   util.NewLockedItem(lastINITVoteproof),
		lastACCEPTVoteproof: util.NewLockedItem(lastACCEPTVoteproof),
	}, nil
}

func (ls *Localstate) Storage() Storage {
	return ls.storage
}

func (ls *Localstate) Node() *LocalNode {
	return ls.node
}

func (ls *Localstate) Policy() *LocalPolicy {
	return ls.policy
}

func (ls *Localstate) Nodes() *NodesState {
	return ls.nodes
}

func (ls *Localstate) LastBlock() block.Block {
	v := ls.lastBlock.Value()
	if v == nil {
		return nil
	}

	return v.(block.Block)
}

// NOTE for debugging and testing only

func (ls *Localstate) SetLastBlock(blk block.Block) error {
	_ = ls.lastBlock.SetValue(blk)

	return nil
}

func (ls *Localstate) LastINITVoteproof() base.Voteproof {
	voteproof := ls.lastINITVoteproof.Value()
	if voteproof == nil {
		return nil
	}

	return voteproof.(base.Voteproof)
}

func (ls *Localstate) SetLastINITVoteproof(voteproof base.Voteproof) error {
	_ = ls.lastINITVoteproof.SetValue(voteproof)

	return nil
}

func (ls *Localstate) LastACCEPTVoteproof() base.Voteproof {
	v := ls.lastACCEPTVoteproof.Value()
	if v == nil {
		return nil
	}

	return v.(base.Voteproof)
}

func (ls *Localstate) SetLastACCEPTVoteproof(voteproof base.Voteproof) error {
	_ = ls.lastACCEPTVoteproof.SetValue(voteproof)

	return nil
}

func (ls *Localstate) Seal(h valuehash.Hash) (seal.Seal, error) {
	if ls.storage != nil {
		return ls.storage.Seal(h)
	}

	return nil, nil
}
