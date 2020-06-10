package base

import (
	"time"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/isvalid"
)

type PolicyOperationBody interface {
	hint.Hinter
	valuehash.Hasher
	util.Byter
	isvalid.IsValider
	ThresholdRatio() ThresholdRatio
	TimeoutWaitingProposal() time.Duration
	IntervalBroadcastingINITBallot() time.Duration
	IntervalBroadcastingProposal() time.Duration
	WaitBroadcastingACCEPTBallot() time.Duration
	IntervalBroadcastingACCEPTBallot() time.Duration
	NumberOfActingSuffrageNodes() uint
	TimespanValidBallot() time.Duration
	TimeoutProcessProposal() time.Duration
}
