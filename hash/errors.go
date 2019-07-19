package hash

import "github.com/spikeekips/mitum/common"

const (
	HashFailedErrorCode common.ErrorCode = iota + 1
	EmptyHashErrorCode
	InvalidHashInputErrorCode
)

var (
	HashFailedError       = common.NewError("hash", HashFailedErrorCode, "failed to make hash")
	EmptyHashError        = common.NewError("hash", EmptyHashErrorCode, "hash is empty")
	InvalidHashInputError = common.NewError("hash", InvalidHashInputErrorCode, "invalid hash input value")
)
