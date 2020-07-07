package isaac

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	PolicyOperationBodyV0Type    = hint.MustNewType(0x08, 0x01, "policy-body-v0")
	PolicyOperationBodyV0Hint    = hint.MustHint(PolicyOperationBodyV0Type, "0.0.1")
	SetPolicyOperationFactV0Type = hint.MustNewType(0x08, 0x02, "set-policy-operation-fact-v0")
	SetPolicyOperationFactV0Hint = hint.MustHint(SetPolicyOperationFactV0Type, "0.0.1")
	SetPolicyOperationV0Type     = hint.MustNewType(0x08, 0x03, "set-policy-operation-v0")
	SetPolicyOperationV0Hint     = hint.MustHint(SetPolicyOperationV0Type, "0.0.1")
)

const PolicyOperationKey = "network_policy"

func DefaultPolicy() PolicyOperationBodyV0 {
	return PolicyOperationBodyV0{
		// NOTE default threshold ratio assumes only one node exists, it means the network is just booted.
		thresholdRatio:                   base.ThresholdRatio(100),
		timeoutWaitingProposal:           time.Second * 5,
		intervalBroadcastingINITBallot:   time.Second * 1,
		intervalBroadcastingProposal:     time.Second * 1,
		waitBroadcastingACCEPTBallot:     time.Second * 2,
		intervalBroadcastingACCEPTBallot: time.Second * 1,
		numberOfActingSuffrageNodes:      uint(1),
		timespanValidBallot:              time.Minute * 1,
		timeoutProcessProposal:           time.Second * 30,
	}
}

type PolicyOperationBodyV0 struct {
	thresholdRatio                   base.ThresholdRatio
	timeoutWaitingProposal           time.Duration
	intervalBroadcastingINITBallot   time.Duration
	intervalBroadcastingProposal     time.Duration
	waitBroadcastingACCEPTBallot     time.Duration
	intervalBroadcastingACCEPTBallot time.Duration
	numberOfActingSuffrageNodes      uint
	timespanValidBallot              time.Duration
	timeoutProcessProposal           time.Duration
}

func (po PolicyOperationBodyV0) Hint() hint.Hint {
	return PolicyOperationBodyV0Hint
}

func (po PolicyOperationBodyV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		util.Float64ToBytes(po.thresholdRatio.Float64()),
		util.DurationToBytes(po.timeoutWaitingProposal),
		util.DurationToBytes(po.intervalBroadcastingINITBallot),
		util.DurationToBytes(po.intervalBroadcastingProposal),
		util.DurationToBytes(po.waitBroadcastingACCEPTBallot),
		util.DurationToBytes(po.intervalBroadcastingACCEPTBallot),
		util.UintToBytes(po.numberOfActingSuffrageNodes),
		util.DurationToBytes(po.timespanValidBallot),
		util.DurationToBytes(po.timeoutProcessProposal),
	)
}

func (po PolicyOperationBodyV0) Hash() valuehash.Hash {
	return valuehash.NewSHA256(po.Bytes())
}

func (po PolicyOperationBodyV0) IsValid([]byte) error {
	for k, d := range map[string]time.Duration{
		"TimeoutWaitingProposal":           po.timeoutWaitingProposal,
		"IntervalBroadcastingINITBallot":   po.intervalBroadcastingINITBallot,
		"IntervalBroadcastingProposal":     po.intervalBroadcastingProposal,
		"WaitBroadcastingACCEPTBallot":     po.waitBroadcastingACCEPTBallot,
		"IntervalBroadcastingACCEPTBallot": po.intervalBroadcastingACCEPTBallot,
		"TimespanValidBallot":              po.timespanValidBallot,
		"TimeoutProcessProposal":           po.timeoutProcessProposal,
	} {
		if d < 0 {
			return xerrors.Errorf("%s is too narrow; duration=%v", k, d)
		}
	}

	if po.numberOfActingSuffrageNodes < 1 {
		return xerrors.Errorf("numberOfActingSuffrageNodes must be over 0; %d", po.numberOfActingSuffrageNodes)
	}

	return po.thresholdRatio.IsValid(nil)
}

func (po PolicyOperationBodyV0) ThresholdRatio() base.ThresholdRatio {
	return po.thresholdRatio
}

func (po PolicyOperationBodyV0) SetThresholdRatio(v base.ThresholdRatio) PolicyOperationBodyV0 {
	po.thresholdRatio = v
	return po
}

func (po PolicyOperationBodyV0) TimeoutWaitingProposal() time.Duration {
	return po.timeoutWaitingProposal
}

func (po PolicyOperationBodyV0) SetTimeoutWaitingProposal(v time.Duration) PolicyOperationBodyV0 {
	po.timeoutWaitingProposal = v
	return po
}

func (po PolicyOperationBodyV0) IntervalBroadcastingINITBallot() time.Duration {
	return po.intervalBroadcastingINITBallot
}

func (po PolicyOperationBodyV0) SetIntervalBroadcastingINITBallot(v time.Duration) PolicyOperationBodyV0 {
	po.intervalBroadcastingINITBallot = v
	return po
}

func (po PolicyOperationBodyV0) IntervalBroadcastingProposal() time.Duration {
	return po.intervalBroadcastingProposal
}

func (po PolicyOperationBodyV0) SetIntervalBroadcastingProposal(v time.Duration) PolicyOperationBodyV0 {
	po.intervalBroadcastingProposal = v
	return po
}

func (po PolicyOperationBodyV0) WaitBroadcastingACCEPTBallot() time.Duration {
	return po.waitBroadcastingACCEPTBallot
}

func (po PolicyOperationBodyV0) SetWaitBroadcastingACCEPTBallot(v time.Duration) PolicyOperationBodyV0 {
	po.waitBroadcastingACCEPTBallot = v
	return po
}

func (po PolicyOperationBodyV0) IntervalBroadcastingACCEPTBallot() time.Duration {
	return po.intervalBroadcastingACCEPTBallot
}

func (po PolicyOperationBodyV0) SetIntervalBroadcastingACCEPTBallot(v time.Duration) PolicyOperationBodyV0 {
	po.intervalBroadcastingACCEPTBallot = v
	return po
}

func (po PolicyOperationBodyV0) NumberOfActingSuffrageNodes() uint {
	return po.numberOfActingSuffrageNodes
}

func (po PolicyOperationBodyV0) SetNumberOfActingSuffrageNodes(v uint) PolicyOperationBodyV0 {
	po.numberOfActingSuffrageNodes = v
	return po
}

func (po PolicyOperationBodyV0) TimespanValidBallot() time.Duration {
	return po.timespanValidBallot
}

func (po PolicyOperationBodyV0) SetTimespanValidBallot(v time.Duration) PolicyOperationBodyV0 {
	po.timespanValidBallot = v
	return po
}

func (po PolicyOperationBodyV0) TimeoutProcessProposal() time.Duration {
	return po.timeoutProcessProposal
}

func (po PolicyOperationBodyV0) SetTimeoutProcessProposal(v time.Duration) PolicyOperationBodyV0 {
	po.timeoutProcessProposal = v
	return po
}

type SetPolicyOperationFactV0 struct {
	PolicyOperationBodyV0
	token []byte
}

func NewSetPolicyOperationFactV0(
	token []byte,
	policies PolicyOperationBodyV0,
) (SetPolicyOperationFactV0, error) {
	return SetPolicyOperationFactV0{
		PolicyOperationBodyV0: policies,
		token:                 token,
	}, nil
}

func (spof SetPolicyOperationFactV0) IsValid([]byte) error {
	if err := spof.Hint().IsValid(nil); err != nil {
		return err
	}

	return spof.PolicyOperationBodyV0.IsValid(nil)
}

func (spof SetPolicyOperationFactV0) Hint() hint.Hint {
	return SetPolicyOperationFactV0Hint
}

func (spof SetPolicyOperationFactV0) Hash() valuehash.Hash {
	return valuehash.NewSHA256(spof.Bytes())
}

func (spof SetPolicyOperationFactV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		spof.token,
		spof.PolicyOperationBodyV0.Bytes(),
	)
}

func (spof SetPolicyOperationFactV0) Token() []byte {
	return spof.token
}

type SetPolicyOperationV0 struct {
	SetPolicyOperationFactV0
	h  valuehash.Hash
	fs operation.FactSign
}

func NewSetPolicyOperationV0(
	signer key.Privatekey,
	token []byte,
	policies PolicyOperationBodyV0,
	b []byte,
) (SetPolicyOperationV0, error) {
	if signer == nil {
		return SetPolicyOperationV0{}, xerrors.Errorf("empty privatekey")
	}

	fact, err := NewSetPolicyOperationFactV0(token, policies)
	if err != nil {
		return SetPolicyOperationV0{}, err
	}

	return NewSetPolicyOperationV0FromFact(signer, fact, b)
}

func NewSetPolicyOperationV0FromFact(signer key.Privatekey, fact SetPolicyOperationFactV0, networkID []byte) (
	SetPolicyOperationV0, error,
) {
	if signer == nil {
		return SetPolicyOperationV0{}, xerrors.Errorf("empty privatekey")
	}

	if err := fact.IsValid(networkID); err != nil {
		return SetPolicyOperationV0{}, err
	}

	var factSignature key.Signature
	if fs, err := signer.Sign(util.ConcatBytesSlice(fact.Hash().Bytes(), networkID)); err != nil {
		return SetPolicyOperationV0{}, err
	} else {
		factSignature = fs
	}

	spo := SetPolicyOperationV0{
		SetPolicyOperationFactV0: fact,
		fs:                       operation.NewBaseFactSign(signer.Publickey(), factSignature),
	}

	if h, err := spo.GenerateHash(); err != nil {
		return SetPolicyOperationV0{}, err
	} else {
		spo.h = h
	}

	return spo, nil
}

func (spo SetPolicyOperationV0) IsValid(networkID []byte) error {
	return operation.IsValidOperation(spo, networkID)
}

func (spo SetPolicyOperationV0) Hint() hint.Hint {
	return SetPolicyOperationV0Hint
}

func (spo SetPolicyOperationV0) Fact() base.Fact {
	return spo.SetPolicyOperationFactV0
}

func (spo SetPolicyOperationV0) Hash() valuehash.Hash {
	return spo.h
}

func (spo SetPolicyOperationV0) Signs() []operation.FactSign {
	return []operation.FactSign{spo.fs}
}

func (spo SetPolicyOperationV0) GenerateHash() (valuehash.Hash, error) {
	return valuehash.NewSHA256(
		util.ConcatBytesSlice(
			spo.Fact().Hash().Bytes(),
			spo.fs.Bytes(),
		),
	), nil
}

func (spo SetPolicyOperationV0) ProcessOperation(
	getState func(key string) (state.StateUpdater, bool, error),
	setState func(state.StateUpdater) error,
) error {
	var value state.HintedValue
	if v, err := state.NewHintedValue(spo.SetPolicyOperationFactV0.PolicyOperationBodyV0); err != nil {
		return err
	} else {
		value = v
	}

	if s, _, err := getState(PolicyOperationKey); err != nil {
		return err
	} else if err := s.SetValue(value); err != nil {
		return err
	} else {
		return setState(s)
	}
}
