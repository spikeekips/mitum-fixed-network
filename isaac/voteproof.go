package isaac

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type Voteproof interface {
	hint.Hinter
	isvalid.IsValider
	util.Byter
	IsFinished() bool
	FinishedAt() time.Time
	IsClosed() bool
	Height() Height
	Round() Round
	Stage() Stage
	Result() VoteResultType
	Majority() operation.Fact
	Ballots() map[Address]valuehash.Hash
	Threshold() Threshold
}

type VoteResultType uint8

const (
	VoteResultNotYet VoteResultType = iota
	VoteResultDraw
	VoteResultMajority
)

func (vrt VoteResultType) Bytes() []byte {
	return util.Uint8ToBytes(uint8(vrt))
}

func (vrt VoteResultType) String() string {
	switch vrt {
	case VoteResultNotYet:
		return "NOT-YET"
	case VoteResultDraw:
		return "DRAW"
	case VoteResultMajority:
		return "MAJORITY"
	default:
		return "<unknown VoteResultType>"
	}
}

func (vrt VoteResultType) IsValid([]byte) error {
	switch vrt {
	case VoteResultNotYet, VoteResultDraw, VoteResultMajority:
		return nil
	}

	return isvalid.InvalidError.Errorf("VoteResultType=%d", vrt)
}

func (vrt VoteResultType) MarshalText() ([]byte, error) {
	return []byte(vrt.String()), nil
}

func (vrt *VoteResultType) UnmarshalText(b []byte) error {
	var t VoteResultType
	switch string(b) {
	case "NOT-YET":
		t = VoteResultNotYet
	case "DRAW":
		t = VoteResultDraw
	case "MAJORITY":
		t = VoteResultMajority
	default:
		return xerrors.Errorf("<unknown VoteResultType>")
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
	}, nil, false); err != nil {
		return err
	}

	return vf.signer.Verify(util.ConcatBytesSlice(vf.fact.Bytes(), b), vf.factSignature)
}
