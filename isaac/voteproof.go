package isaac

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/valuehash"
)

var VoteproofType hint.Type = hint.Type{0x04, 0x00}

type Voteproof interface {
	hint.Hinter
	isvalid.IsValider
	Bytes() []byte
	IsFinished() bool
	FinishedAt() time.Time
	IsClosed() bool
	Height() Height
	Round() Round
	Stage() Stage
	Result() VoteproofResultType
	Majority() Fact
	Ballots() map[Address]valuehash.Hash
}

type VoteproofResultType uint8

const (
	VoteproofNotYet VoteproofResultType = iota
	VoteproofDraw
	VoteproofMajority
)

func (vrt VoteproofResultType) String() string {
	switch vrt {
	case VoteproofNotYet:
		return "NOT-YET"
	case VoteproofDraw:
		return "DRAW"
	case VoteproofMajority:
		return "MAJORITY"
	default:
		return "<unknown VoteproofResultType>"
	}
}

func (vrt VoteproofResultType) IsValid([]byte) error {
	switch vrt {
	case VoteproofNotYet, VoteproofDraw, VoteproofMajority:
		return nil
	}

	return isvalid.InvalidError.Wrapf("VoteproofResultType=%d", vrt)
}

func (vrt VoteproofResultType) MarshalText() ([]byte, error) {
	return []byte(vrt.String()), nil
}

func (vrt *VoteproofResultType) UnmarshalText(b []byte) error {
	var t VoteproofResultType
	switch string(b) {
	case "NOT-YET":
		t = VoteproofNotYet
	case "DRAW":
		t = VoteproofDraw
	case "MAJORITY":
		t = VoteproofMajority
	default:
		return xerrors.Errorf("<unknown VoteproofResultType>")
	}

	*vrt = t

	return nil
}

type VoteproofNodeFact struct {
	fact          valuehash.Hash
	factSignature key.Signature
	signer        key.Publickey
}

func (vf VoteproofNodeFact) IsValid(b []byte) error {
	if err := isvalid.Check([]isvalid.IsValider{
		vf.fact,
		vf.factSignature,
		vf.signer,
	}, b, false); err != nil {
		return err
	}

	return vf.signer.Verify(vf.fact.Bytes(), vf.factSignature)
}
