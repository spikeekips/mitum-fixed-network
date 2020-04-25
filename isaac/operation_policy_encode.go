package isaac

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util/encoder"
)

func (po *PolicyOperationBodyV0) unpack(
	rawThreshold []float64,
	timeoutWaitingProposal time.Duration,
	intervalBroadcastingINITBallot time.Duration,
	intervalBroadcastingProposal time.Duration,
	waitBroadcastingACCEPTBallot time.Duration,
	intervalBroadcastingACCEPTBallot time.Duration,
	numberOfActingSuffrageNodes uint,
	timespanValidBallot time.Duration,
) error {
	var err error

	var threshold base.Threshold
	if len(rawThreshold) != 2 {
		return xerrors.Errorf("invalid formatted Threshold found: %v", rawThreshold)
	} else if total := rawThreshold[0]; total < 0 {
		return xerrors.Errorf("invalid total number of Threshold found: %v", rawThreshold)
	} else if percent := rawThreshold[1]; percent < 0 {
		return xerrors.Errorf("invalid percent number of Threshold found: %v", rawThreshold)
	} else if threshold, err = base.NewThreshold(uint(total), percent); err != nil {
		return err
	}

	po.Threshold = threshold
	po.TimeoutWaitingProposal = timeoutWaitingProposal
	po.IntervalBroadcastingINITBallot = intervalBroadcastingINITBallot
	po.IntervalBroadcastingProposal = intervalBroadcastingProposal
	po.WaitBroadcastingACCEPTBallot = waitBroadcastingACCEPTBallot
	po.IntervalBroadcastingACCEPTBallot = intervalBroadcastingACCEPTBallot
	po.NumberOfActingSuffrageNodes = numberOfActingSuffrageNodes
	po.TimespanValidBallot = timespanValidBallot

	return nil
}

func (spo *SetPolicyOperationV0) unpack(
	enc encoder.Encoder,
	bHash,
	bFactHash []byte,
	factSignature key.Signature,
	bSigner,
	token,
	bPolicies []byte,
) error {
	var err error

	var h, factHash valuehash.Hash
	if h, err = valuehash.Decode(enc, bHash); err != nil {
		return err
	}
	if factHash, err = valuehash.Decode(enc, bFactHash); err != nil {
		return err
	}
	var signer key.Publickey
	if signer, err = key.DecodePublickey(enc, bSigner); err != nil {
		return err
	}

	var body PolicyOperationBodyV0
	if err := enc.Decode(bPolicies, &body); err != nil {
		return err
	}

	spo.h = h
	spo.factHash = factHash
	spo.factSignature = factSignature
	spo.SetPolicyOperationFactV0 = SetPolicyOperationFactV0{
		PolicyOperationBodyV0: body,
		signer:                signer,
		token:                 token,
	}

	return nil
}
