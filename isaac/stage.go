package isaac

import (
	"encoding/binary"
	"encoding/json"

	"golang.org/x/xerrors"
)

type Stage uint

const (
	StageNone Stage = iota + 33
	StageINIT
	StageSIGN
	StageACCEPT
	StageALLCONFIRM
)

func (s Stage) String() string {
	switch s {
	case StageINIT:
		return "INIT"
	case StageSIGN:
		return "SIGN"
	case StageACCEPT:
		return "ACCEPT"
	case StageALLCONFIRM:
		return "ALLCONFIRM"
	default:
		return "<unknown stage>"
	}
}

func (s Stage) MarshalBinary() ([]byte, error) {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, uint32(s))
	return b, nil
}

func (s *Stage) UnmarshalBinary(b []byte) error {
	u := binary.LittleEndian.Uint32(b)

	*s = Stage(u)

	return nil
}

func (s Stage) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s Stage) IsValid() error {
	switch s {
	case StageINIT:
	case StageSIGN:
	case StageACCEPT:
	//case StageALLCONFIRM:
	default:
		return xerrors.Errorf("unknown stage")
	}

	return nil
}

func (s Stage) Next() Stage {
	switch s {
	case StageINIT:
		return StageSIGN
	case StageSIGN:
		return StageACCEPT
	case StageACCEPT:
		return StageINIT
	default:
		panic(InvalidStageError)
	}
}

func (s Stage) CanVote() bool {
	switch s {
	case StageINIT:
	case StageSIGN:
	case StageACCEPT:
	default:
		return false
	}

	return true
}
