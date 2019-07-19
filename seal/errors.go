package seal

import "github.com/spikeekips/mitum/common"

const (
	InvalidSealErrorCode common.ErrorCode = iota + 1
)

var (
	InvalidSealError = common.NewError("seal", InvalidSealErrorCode, "invalid seal")
)
