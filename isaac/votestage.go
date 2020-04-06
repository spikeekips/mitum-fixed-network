package isaac

import (
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/isvalid"
	"github.com/spikeekips/mitum/util"
)

type Stage uint8

const (
	_ Stage = iota
	StageINIT
	StageSIGN
	StageACCEPT
	StageProposal
)

func (st Stage) Bytes() []byte {
	return util.Uint8ToBytes(uint8(st))
}

func (st Stage) String() string {
	switch st {
	case StageINIT:
		return "INIT"
	case StageSIGN:
		return "SIGN"
	case StageACCEPT:
		return "ACCEPT"
	case StageProposal:
		return "PROPOSAL"
	default:
		return "<unknown stage>"
	}
}

func (st Stage) IsValid([]byte) error {
	switch st {
	case StageINIT, StageSIGN, StageACCEPT, StageProposal:
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
	case "SIGN":
		t = StageSIGN
	case "ACCEPT":
		t = StageACCEPT
	case "PROPOSAL":
		t = StageProposal
	default:
		return xerrors.Errorf("<unknown stage>")
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
