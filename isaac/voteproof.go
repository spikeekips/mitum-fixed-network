package isaac

import (
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/valuehash"
)

var (
	VoteProofType hint.Type = hint.Type([2]byte{0x04, 0x00})
)

type VoteProof interface {
	hint.Hinter
	isvalid.IsValider
	Bytes() []byte
	IsFinished() bool
	IsClosed() bool
	Height() Height
	Round() Round
	Stage() Stage
	Result() VoteProofResultType
	Majority() Fact
	Ballots() map[Address]valuehash.Hash
	CompareWithBlock(Block) error
}

type VoteProofResultType uint8

const (
	VoteProofNotYet VoteProofResultType = iota
	VoteProofDraw
	VoteProofMajority
)

func (vrt VoteProofResultType) String() string {
	switch vrt {
	case VoteProofNotYet:
		return "NOT-YET"
	case VoteProofDraw:
		return "DRAW"
	case VoteProofMajority:
		return "MAJORITY"
	default:
		return "<unknown VoteProofResultType>"
	}
}

func (vrt VoteProofResultType) IsValid([]byte) error {
	switch vrt {
	case VoteProofNotYet, VoteProofDraw, VoteProofMajority:
		return nil
	}

	return InvalidError.Wrapf("VoteProofResultType=%d", vrt)
}

func (vrt VoteProofResultType) MarshalText() ([]byte, error) {
	return []byte(vrt.String()), nil
}

type VoteProofNodeFact struct {
	fact          valuehash.Hash
	factSignature key.Signature
	signer        key.Publickey
}

func (vf VoteProofNodeFact) IsValid(b []byte) error {
	// TODO check,
	// - signer is valid Ballot.Signer()?
	if err := isvalid.Check([]isvalid.IsValider{
		vf.fact,
		vf.factSignature,
		vf.signer,
	}, b); err != nil {
		return err
	}

	return vf.signer.Verify(vf.fact.Bytes(), vf.factSignature)
}
