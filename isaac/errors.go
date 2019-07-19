package isaac

import "github.com/spikeekips/mitum/common"

const (
	InvalidStageErrorCode common.ErrorCode = iota + 1
	InvalidBallotErrorCode
)

var (
	InvalidStageError  = common.NewError("isaac", InvalidStageErrorCode, "invalid stage")
	InvalidBallotError = common.NewError("isaac", InvalidBallotErrorCode, "invalid ballot")
)
