package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/ballot"
	"github.com/spikeekips/mitum/base/block"
	"github.com/spikeekips/mitum/base/valuehash"
)

func NewINITBallotV0(
	localstate *Localstate,
	height base.Height,
	round base.Round,
	previousBlock valuehash.Hash,
	previousRound base.Round,
	voteproof base.Voteproof,
) (ballot.INITBallotV0, error) {
	ib := ballot.NewINITBallotV0(
		localstate.Node().Address(),
		height,
		round,
		previousBlock,
		previousRound,
		voteproof,
	)

	if err := ib.Sign(localstate.Node().Privatekey(), localstate.Policy().NetworkID()); err != nil {
		return ballot.INITBallotV0{}, err
	}

	return ib, nil
}

func NewINITBallotV0FromLocalstate(localstate *Localstate, round base.Round) (ballot.INITBallotV0, error) {
	lastBlock := localstate.LastBlock()
	if lastBlock == nil {
		return ballot.INITBallotV0{}, xerrors.Errorf("lastBlock is empty")
	}

	var voteproof base.Voteproof
	if round == 0 {
		voteproof = localstate.LastACCEPTVoteproof()
	} else {
		voteproof = localstate.LastINITVoteproof()
	}

	ib := ballot.NewINITBallotV0(
		localstate.Node().Address(),
		lastBlock.Height()+1,
		round,
		lastBlock.Hash(),
		lastBlock.Round(),
		voteproof,
	)

	if err := ib.Sign(localstate.Node().Privatekey(), localstate.Policy().NetworkID()); err != nil {
		return ballot.INITBallotV0{}, err
	}

	return ib, nil
}

func NewProposal(
	localstate *Localstate,
	height base.Height,
	round base.Round,
	operations []valuehash.Hash,
	seals []valuehash.Hash,
	networkID []byte,
) (ballot.Proposal, error) {
	pr := ballot.NewProposalV0(
		localstate.Node().Address(),
		height,
		round,
		operations,
		seals,
	)

	if err := pr.Sign(localstate.Node().Privatekey(), networkID); err != nil {
		return ballot.ProposalV0{}, err
	}

	return pr, nil
}

func NewProposalFromLocalstate(
	localstate *Localstate,
	round base.Round,
	operations []valuehash.Hash,
	seals []valuehash.Hash,
) (ballot.Proposal, error) {
	lastBlock := localstate.LastBlock()
	if lastBlock == nil {
		return ballot.ProposalV0{}, xerrors.Errorf("lastBlock is empty")
	}

	pr := ballot.NewProposalV0(
		localstate.Node().Address(),
		lastBlock.Height()+1,
		round,
		operations,
		seals,
	)

	if err := pr.Sign(localstate.Node().Privatekey(), localstate.Policy().NetworkID()); err != nil {
		return ballot.ProposalV0{}, err
	}

	return pr, nil
}

func NewSIGNBallotV0FromLocalstate(
	localstate *Localstate, round base.Round, newBlock block.Block,
) (ballot.SIGNBallotV0, error) {
	lastBlock := localstate.LastBlock()
	if lastBlock == nil {
		return ballot.SIGNBallotV0{}, xerrors.Errorf("lastBlock is empty")
	}

	sb := ballot.NewSIGNBallotV0(
		localstate.Node().Address(),
		lastBlock.Height()+1,
		round,
		newBlock.Proposal(),
		newBlock.Hash(),
	)

	if err := sb.Sign(localstate.Node().Privatekey(), localstate.Policy().NetworkID()); err != nil {
		return ballot.SIGNBallotV0{}, err
	}

	return sb, nil
}

func NewACCEPTBallotV0(
	localstate *Localstate,
	height base.Height,
	round base.Round,
	newBlock block.Block,
	initVoteproof base.Voteproof,
	networkID []byte,
) (ballot.ACCEPTBallotV0, error) {
	ab := ballot.NewACCEPTBallotV0(
		localstate.Node().Address(),
		height,
		round,
		newBlock.Proposal(),
		newBlock.Hash(),
		initVoteproof,
	)

	if err := ab.Sign(localstate.Node().Privatekey(), networkID); err != nil {
		return ballot.ACCEPTBallotV0{}, err
	}

	return ab, nil
}

func NewACCEPTBallotV0FromLocalstate(
	localstate *Localstate,
	round base.Round,
	newBlock block.Block,
) (ballot.ACCEPTBallotV0, error) {
	lastBlock := localstate.LastBlock()
	if lastBlock == nil {
		return ballot.ACCEPTBallotV0{}, xerrors.Errorf("lastBlock is empty")
	}

	ab := ballot.NewACCEPTBallotV0(
		localstate.Node().Address(),
		lastBlock.Height()+1,
		round,
		newBlock.Proposal(),
		newBlock.Hash(),
		localstate.LastINITVoteproof(),
	)

	if err := ab.Sign(localstate.Node().Privatekey(), localstate.Policy().NetworkID()); err != nil {
		return ballot.ACCEPTBallotV0{}, err
	}

	return ab, nil
}
