package isaac

import (
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/util/encoder"
	"github.com/spikeekips/mitum/util/valuehash"
)

func (po *PolicyOperationBodyV0) unpack(
	thresholdRatio base.ThresholdRatio,
	timeoutWaitingProposal time.Duration,
	intervalBroadcastingINITBallot time.Duration,
	intervalBroadcastingProposal time.Duration,
	waitBroadcastingACCEPTBallot time.Duration,
	intervalBroadcastingACCEPTBallot time.Duration,
	numberOfActingSuffrageNodes uint,
	timespanValidBallot,
	timeoutProcessProposal time.Duration,
) error {
	po.thresholdRatio = thresholdRatio
	po.timeoutWaitingProposal = timeoutWaitingProposal
	po.intervalBroadcastingINITBallot = intervalBroadcastingINITBallot
	po.intervalBroadcastingProposal = intervalBroadcastingProposal
	po.waitBroadcastingACCEPTBallot = waitBroadcastingACCEPTBallot
	po.intervalBroadcastingACCEPTBallot = intervalBroadcastingACCEPTBallot
	po.numberOfActingSuffrageNodes = numberOfActingSuffrageNodes
	po.timespanValidBallot = timespanValidBallot
	po.timeoutProcessProposal = timeoutProcessProposal

	return nil
}

func (spo *SetPolicyOperationV0) unpack(
	enc encoder.Encoder,
	h valuehash.Hash,
	bfs []byte,
	token,
	bPolicies []byte,
) error {
	var body PolicyOperationBodyV0
	if err := enc.Decode(bPolicies, &body); err != nil {
		return err
	}

	var fs operation.FactSign
	if f, err := operation.DecodeFactSign(enc, bfs); err != nil {
		return err
	} else {
		fs = f
	}

	spo.h = h
	spo.fs = fs
	spo.SetPolicyOperationFactV0 = SetPolicyOperationFactV0{
		PolicyOperationBodyV0: body,
		token:                 token,
	}

	return nil
}
