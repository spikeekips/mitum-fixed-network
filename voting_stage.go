package mitum

import "github.com/spikeekips/mitum/errors"

var (
	InvalidStageError = errors.NewError("invalid stage")
)

type Stage uint8

const (
	_ Stage = iota
	StageINIT
	StageSIGN
	StageACCEPT
)

func (st Stage) String() string {
	switch st {
	case StageINIT:
		return "INIT"
	case StageSIGN:
		return "SIGN"
	case StageACCEPT:
		return "ACCEPT"
	default:
		return "<unknown stage>"
	}
}

func (st Stage) IsValid() error {
	switch st {
	case StageINIT, StageSIGN, StageACCEPT:
		return nil
	}

	return InvalidStageError.Wrapf("stage=%d", st)
}
