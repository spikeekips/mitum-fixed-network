package isaac

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
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
		// NOTE default threshold assumes only one node exists, it means the network is just booted.
		Threshold:                        base.MustNewThreshold(1, 100),
		TimeoutWaitingProposal:           time.Second * 5,
		IntervalBroadcastingINITBallot:   time.Second * 1,
		IntervalBroadcastingProposal:     time.Second * 1,
		WaitBroadcastingACCEPTBallot:     time.Second * 2,
		IntervalBroadcastingACCEPTBallot: time.Second * 1,
		NumberOfActingSuffrageNodes:      uint(1),
		TimespanValidBallot:              time.Minute * 1,
	}
}

type PolicyOperationBodyV0 struct {
	Threshold                        base.Threshold `json:"threshold"`
	TimeoutWaitingProposal           time.Duration  `json:"timeout_waiting_proposal"`
	IntervalBroadcastingINITBallot   time.Duration  `json:"interval_broadcasting_init_ballot"`
	IntervalBroadcastingProposal     time.Duration  `json:"interval_broadcasting_proposal"`
	WaitBroadcastingACCEPTBallot     time.Duration  `json:"wait_broadcasting_accept_ballot"`
	IntervalBroadcastingACCEPTBallot time.Duration  `json:"interval_broadcasting_accept_ballot"`
	NumberOfActingSuffrageNodes      uint           `json:"number_of_acting_suffrage_nodes"`
	TimespanValidBallot              time.Duration  `json:"timespan_valid_ballot"`
}

func (po PolicyOperationBodyV0) Hint() hint.Hint {
	return PolicyOperationBodyV0Hint
}

func (po PolicyOperationBodyV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		po.Threshold.Bytes(),
		util.DurationToBytes(po.TimeoutWaitingProposal),
		util.DurationToBytes(po.IntervalBroadcastingINITBallot),
		util.DurationToBytes(po.IntervalBroadcastingProposal),
		util.DurationToBytes(po.WaitBroadcastingACCEPTBallot),
		util.DurationToBytes(po.IntervalBroadcastingACCEPTBallot),
		util.UintToBytes(po.NumberOfActingSuffrageNodes),
		util.DurationToBytes(po.TimespanValidBallot),
	)
}

func (po PolicyOperationBodyV0) Hash() valuehash.Hash {
	return valuehash.NewSHA256(po.Bytes())
}

func (po PolicyOperationBodyV0) IsValid([]byte) error {
	for k, d := range map[string]time.Duration{
		"TimeoutWaitingProposal":           po.TimeoutWaitingProposal,
		"IntervalBroadcastingINITBallot":   po.IntervalBroadcastingINITBallot,
		"IntervalBroadcastingProposal":     po.IntervalBroadcastingProposal,
		"WaitBroadcastingACCEPTBallot":     po.WaitBroadcastingACCEPTBallot,
		"IntervalBroadcastingACCEPTBallot": po.IntervalBroadcastingACCEPTBallot,
		"TimespanValidBallot":              po.TimespanValidBallot,
	} {
		if d < 0 {
			return xerrors.Errorf("%s is too narrow; duration=%v", k, d)
		}
	}

	if po.NumberOfActingSuffrageNodes < 1 {
		return xerrors.Errorf("NumberOfActingSuffrageNodes must be over 0; %d", po.NumberOfActingSuffrageNodes)
	}

	if err := po.Threshold.IsValid(nil); err != nil {
		return err
	}

	return nil
}

type SetPolicyOperationFactV0 struct {
	PolicyOperationBodyV0
	signer key.Publickey
	token  []byte
}

func (spof SetPolicyOperationFactV0) IsValid([]byte) error {
	if spof.signer == nil {
		return xerrors.Errorf("fact has empty signer")
	}
	if err := spof.Hint().IsValid(nil); err != nil {
		return err
	}

	if err := spof.PolicyOperationBodyV0.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (spof SetPolicyOperationFactV0) Hint() hint.Hint {
	return SetPolicyOperationFactV0Hint
}

func (spof SetPolicyOperationFactV0) Hash() valuehash.Hash {
	return valuehash.NewSHA256(spof.Bytes())
}

func (spof SetPolicyOperationFactV0) Bytes() []byte {
	return util.ConcatBytesSlice(
		[]byte(spof.signer.String()),
		spof.token,
		spof.PolicyOperationBodyV0.Bytes(),
	)
}

func (spof SetPolicyOperationFactV0) Signer() key.Publickey {
	return spof.signer
}

func (spof SetPolicyOperationFactV0) Token() []byte {
	return spof.token
}

type SetPolicyOperationV0 struct {
	SetPolicyOperationFactV0
	h             valuehash.Hash
	factHash      valuehash.Hash
	factSignature key.Signature
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

	fact := SetPolicyOperationFactV0{
		PolicyOperationBodyV0: policies,
		signer:                signer.Publickey(),
		token:                 token,
	}
	factHash := fact.Hash()
	var factSignature key.Signature
	if fs, err := signer.Sign(util.ConcatBytesSlice(factHash.Bytes(), b)); err != nil {
		return SetPolicyOperationV0{}, err
	} else {
		factSignature = fs
	}

	spo := SetPolicyOperationV0{
		SetPolicyOperationFactV0: fact,
		factHash:                 factHash,
		factSignature:            factSignature,
	}

	if h, err := spo.GenerateHash(); err != nil {
		return SetPolicyOperationV0{}, err
	} else {
		spo.h = h
	}

	return spo, nil
}

func (spo SetPolicyOperationV0) IsValid(b []byte) error {
	if err := operation.IsValidOperation(spo, b); err != nil {
		return err
	}

	return nil
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

func (spo SetPolicyOperationV0) GenerateHash() (valuehash.Hash, error) {
	return valuehash.NewSHA256(util.ConcatBytesSlice(spo.factHash.Bytes(), spo.factSignature.Bytes())), nil
}

func (spo SetPolicyOperationV0) FactHash() valuehash.Hash {
	return spo.factHash
}

func (spo SetPolicyOperationV0) FactSignature() key.Signature {
	return spo.factSignature
}

func (spo SetPolicyOperationV0) ProcessOperation(
	getState func(key string) (state.StateUpdater, error),
	setState func(state.StateUpdater) error,
) (state.StateUpdater, error) {
	value, err := state.NewHintedValue(spo.SetPolicyOperationFactV0.PolicyOperationBodyV0)
	if err != nil {
		return nil, err
	}

	var st state.StateUpdater
	if s, err := getState(PolicyOperationKey); err != nil {
		return nil, err
	} else if err := s.SetValue(value); err != nil {
		return nil, err
	} else {
		st = s
	}

	return st, setState(st)
}
