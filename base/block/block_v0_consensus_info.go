package block

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

type ConsensusInfoV0 struct {
	hint.BaseHinter
	initVoteproof   base.Voteproof
	acceptVoteproof base.Voteproof
	suffrageInfo    SuffrageInfo
	sfs             base.SignedBallotFact
}

func (bc ConsensusInfoV0) IsValid(networkID []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		bc.BaseHinter,
		bc.initVoteproof,
		bc.acceptVoteproof,
		bc.suffrageInfo,
		bc.sfs,
	}, networkID, false); err != nil {
		return err
	}

	if bc.sfs != nil && bc.sfs.Fact().Hint().Type() != base.ProposalFactType {
		return isvalid.InvalidError.Errorf("proposal does not have proposal fact type; %q", bc.sfs.Fact().Hint().Type())
	}

	if bc.initVoteproof.Stage() != base.StageINIT {
		return errors.Errorf("invalid initVoteproof, %v found in ConsensusInfo", bc.initVoteproof.Stage())
	} else if bc.acceptVoteproof.Stage() != base.StageACCEPT {
		return errors.Errorf("invalid acceptVoteproof, %v found in ConsensusInfo", bc.acceptVoteproof.Stage())
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

func (ConsensusInfoV0) isValidVoteproof(
	_ []byte,
	sn map[base.Address]base.Node,
	voteproof base.Voteproof,
) error {
	for i := range voteproof.Votes() {
		nf := voteproof.Votes()[i]
		if node, found := sn[nf.FactSign().Node()]; !found {
			return errors.Errorf("unknown node, %s voted in %v voteproof.Votes()",
				nf.FactSign().Node(), voteproof.Stage())
		} else if !nf.FactSign().Signer().Equal(node.Publickey()) {
			return errors.Errorf("node, %s has invalid Publickey in %v voteproof",
				nf.FactSign().Node(), voteproof.Stage())
		}
	}

	return nil
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

func (bc ConsensusInfoV0) Proposal() base.SignedBallotFact {
	return bc.sfs
}

type SuffrageInfoV0 struct {
	hint.BaseHinter
	proposer base.Address
	nodes    []base.Node
}

func NewSuffrageInfoV0(proposer base.Address, nodes []base.Node) SuffrageInfoV0 {
	return SuffrageInfoV0{
		BaseHinter: hint.NewBaseHinter(SuffrageInfoV0Hint),
		proposer:   proposer,
		nodes:      nodes,
	}
}

func (si SuffrageInfoV0) Proposer() base.Address {
	return si.proposer
}

func (si SuffrageInfoV0) Nodes() []base.Node {
	return si.nodes
}

func (si SuffrageInfoV0) IsValid([]byte) error {
	if err := si.BaseHinter.IsValid(nil); err != nil {
		return err
	}

	var found bool
	vs := []isvalid.IsValider{si.BaseHinter, si.proposer}
	for _, n := range si.nodes {
		if !found && si.proposer.Equal(n.Address()) {
			found = true
		}

		vs = append(vs, n.Address(), n.Publickey())
	}

	if !found {
		return errors.Errorf("proposer not found in nodes")
	}

	return isvalid.Check(vs, nil, false)
}
