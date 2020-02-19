package isaac

import (
	"time"

	"golang.org/x/xerrors"

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
	FinishedAt() time.Time
	IsClosed() bool
	Height() Height
	Round() Round
	Stage() Stage
	Result() VoteProofResultType
	Majority() Fact
	Ballots() map[Address]valuehash.Hash
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

	return isvalid.InvalidError.Wrapf("VoteProofResultType=%d", vrt)
}

func (vrt VoteProofResultType) MarshalText() ([]byte, error) {
	return []byte(vrt.String()), nil
}

func (vrt *VoteProofResultType) UnmarshalText(b []byte) error {
	var t VoteProofResultType
	switch string(b) {
	case "NOT-YET":
		t = VoteProofNotYet
	case "DRAW":
		t = VoteProofDraw
	case "MAJORITY":
		t = VoteProofMajority
	default:
		return xerrors.Errorf("<unknown VoteProofResultType>")
	}

	*vrt = t

	return nil
}

type VoteProofNodeFact struct {
	fact          valuehash.Hash
	factSignature key.Signature
	signer        key.Publickey
}

func (vf VoteProofNodeFact) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		vf.fact,
		vf.factSignature,
		vf.signer,
	}, b, false); err != nil {
		return err
	}

	return vf.signer.Verify(vf.fact.Bytes(), vf.factSignature)
}
