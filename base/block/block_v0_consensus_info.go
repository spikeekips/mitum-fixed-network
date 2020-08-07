package block

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

type ConsensusInfoV0 struct {
	initVoteproof   base.Voteproof
	acceptVoteproof base.Voteproof
	suffrageInfo    SuffrageInfo
	proposal        ballot.Proposal
}

func (bc ConsensusInfoV0) IsValid(networkID []byte) error {
	if err := isvalid.Check(
		[]isvalid.IsValider{
			bc.initVoteproof,
			bc.acceptVoteproof,
			bc.suffrageInfo,
			bc.proposal,
		},
		networkID, false); err != nil {
		return err
	}

	if bc.initVoteproof.Stage() != base.StageINIT {
		return xerrors.Errorf("invalid initVoteproof, %v found in ConsensusInfo", bc.initVoteproof.Stage())
	} else if bc.acceptVoteproof.Stage() != base.StageACCEPT {
		return xerrors.Errorf("invalid acceptVoteproof, %v found in ConsensusInfo", bc.acceptVoteproof.Stage())
	}

	sn := map[base.Address]base.Node{}
	for _, node := range bc.suffrageInfo.Nodes() {
		sn[node.Address()] = node
	}

	if err := bc.isValidVoteproof(nil, sn, bc.initVoteproof); err != nil {
		return err
	} else if err := bc.isValidVoteproof(nil, sn, bc.acceptVoteproof); err != nil {
		return err
	}

	return nil
}

func (bc ConsensusInfoV0) isValidVoteproof(
	_ []byte,
	sn map[base.Address]base.Node,
	voteproof base.Voteproof,
) error {
	for i := range voteproof.Votes() {
		nf := voteproof.Votes()[i]
		if node, found := sn[nf.Node()]; !found {
			return xerrors.Errorf("unknown node, %s voted in %v voteproof.Votes()", nf.Node(), voteproof.Stage())
		} else if !nf.Signer().Equal(node.Publickey()) {
			return xerrors.Errorf("node, %s has invalid Publickey in %v voteproof", nf.Node(), voteproof.Stage())
		}
	}

	return nil
}

func (bc ConsensusInfoV0) Hint() hint.Hint {
	return BlockConsensusInfoV0Hint
}

func (bc ConsensusInfoV0) INITVoteproof() base.Voteproof {
	return bc.initVoteproof
}

func (bc ConsensusInfoV0) ACCEPTVoteproof() base.Voteproof {
	return bc.acceptVoteproof
}

func (bc ConsensusInfoV0) SuffrageInfo() SuffrageInfo {
	return bc.suffrageInfo
}

func (bc ConsensusInfoV0) Proposal() ballot.Proposal {
	return bc.proposal
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

func (si SuffrageInfoV0) Hint() hint.Hint {
	return SuffrageInfoV0Hint
}

func (si SuffrageInfoV0) Proposer() base.Address {
	return si.proposer
}

func (si SuffrageInfoV0) Nodes() []base.Node {
	return si.nodes
}

func (si SuffrageInfoV0) IsValid([]byte) error {
	var found bool
	vs := []isvalid.IsValider{si.proposer}
	for _, n := range si.nodes {
		if !found && si.proposer.Equal(n.Address()) {
			found = true
		}

		vs = append(vs, n.Address(), n.Publickey())
	}

	if !found {
		return xerrors.Errorf("proposer not found in nodes")
	}

	return isvalid.Check(vs, nil, false)
}
