package block

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

type BlockConsensusInfoV0 struct {
	initVoteproof   base.Voteproof
	acceptVoteproof base.Voteproof
	suffrageInfo    SuffrageInfo
}

func (bc BlockConsensusInfoV0) IsValid([]byte) error {
	return isvalid.Check(
		[]isvalid.IsValider{
			bc.initVoteproof,
			bc.acceptVoteproof,
			bc.suffrageInfo,
		},
		nil, false,
	)
}

func (bc BlockConsensusInfoV0) Hint() hint.Hint {
	return BlockConsensusInfoV0Hint
}

func (bc BlockConsensusInfoV0) INITVoteproof() base.Voteproof {
	return bc.initVoteproof
}

func (bc BlockConsensusInfoV0) ACCEPTVoteproof() base.Voteproof {
	return bc.acceptVoteproof
}

func (bc BlockConsensusInfoV0) SuffrageInfo() SuffrageInfo {
	return bc.suffrageInfo
}

type SuffrageInfoV0 struct {
	proposer base.Address
	nodes    []base.Node
}

func NewSuffrageInfoV0(proposer base.Address, nodes []base.Node) SuffrageInfoV0 {
	return SuffrageInfoV0{
		proposer: proposer,
		nodes:    nodes,
	}
}

func (bn SuffrageInfoV0) Hint() hint.Hint {
	return SuffrageInfoV0Hint
}

func (bn SuffrageInfoV0) Proposer() base.Address {
	return bn.proposer
}

func (bn SuffrageInfoV0) Nodes() []base.Node {
	return bn.nodes
}

func (bn SuffrageInfoV0) IsValid([]byte) error {
	var found bool
	vs := []isvalid.IsValider{bn.proposer}
	for _, n := range bn.nodes {
		if !found && bn.proposer.Equal(n.Address()) {
			found = true
		}

		vs = append(vs, n.Address(), n.Publickey())
	}

	if !found {
		return xerrors.Errorf("proposer not found in nodes")
	}

	return isvalid.Check(vs, nil, false)
}
