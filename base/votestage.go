package base

import (
	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/isvalid"
)

type Stage uint8

const (
	_ Stage = iota
	StageINIT
	StageProposal
	StageACCEPT
)

func (st Stage) Bytes() []byte {
	return util.Uint8ToBytes(uint8(st))
}

func (st Stage) String() string {
	switch st {
	case StageINIT:
		return "INIT"
	case StageProposal:
		return "PROPOSAL"
	case StageACCEPT:
		return "ACCEPT"
	default:
		return "<unknown stage>"
	}
}

func (st Stage) IsValid([]byte) error {
	switch st {
	case StageINIT, StageACCEPT, StageProposal:
		return nil
	}

	return isvalid.InvalidError.Errorf("stage=%d", st)
}

func (st Stage) MarshalText() ([]byte, error) {
	return []byte(st.String()), nil
}

func (st *Stage) UnmarshalText(b []byte) error {
	var t Stage
	switch string(b) {
	case "INIT":
		t = StageINIT
	case "PROPOSAL":
		t = StageProposal
	case "ACCEPT":
		t = StageACCEPT
	default:
		return errors.Errorf("<unknown stage>")
	}

	*st = t

	return nil
}

func (st Stage) CanVote() bool {
	switch st {
	case StageINIT, StageACCEPT:
		return true
	default:
		return false
	}
}
