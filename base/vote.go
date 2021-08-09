package base

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/isvalid"
)

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
		return errors.Errorf("<unknown VoteResultType>")
	}

	*vrt = t

	return nil
}
