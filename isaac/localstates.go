package isaac

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

type Localstate struct {
	storage             storage.Storage
	node                *LocalNode
	policy              *LocalPolicy
	nodes               *NodesState
	lastINITVoteproof   *util.LockedItem
	lastACCEPTVoteproof *util.LockedItem
}

func NewLocalstate(st storage.Storage, node *LocalNode, networkID []byte) (*Localstate, error) {
	// load last states from storage.
	var lastINITVoteproof base.Voteproof
	var lastACCEPTVoteproof base.Voteproof
	if st != nil {
		if l, err := st.LastBlock(); err != nil {
			if !xerrors.Is(err, storage.NotFoundError) {
				return nil, err
			}
		} else {
			lastINITVoteproof = l.INITVoteproof()
			lastACCEPTVoteproof = l.ACCEPTVoteproof()
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
		lastINITVoteproof:   util.NewLockedItem(lastINITVoteproof),
		lastACCEPTVoteproof: util.NewLockedItem(lastACCEPTVoteproof),
	}, nil
}

func (ls *Localstate) Storage() storage.Storage {
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
