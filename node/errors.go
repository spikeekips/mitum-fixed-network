package node

import "github.com/spikeekips/mitum/common"

const (
	InvalidStateErrorCode common.ErrorCode = iota + 1
)

var (
	InvalidStateError = common.NewError("node", InvalidStateErrorCode, "invalid node state")
)
