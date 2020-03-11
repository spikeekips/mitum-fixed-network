package isaac

import (
	"time"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/key"
	"github.com/spikeekips/mitum/operation"
	"github.com/spikeekips/mitum/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

var (
	SetPolicyOperationFactV0Type = hint.MustNewType(0x08, 0x01, "set-policy-operation-fact-v0")
	SetPolicyOperationFactV0Hint = hint.MustHint(SetPolicyOperationFactV0Type, "0.0.1")
	SetPolicyOperationV0Type     = hint.MustNewType(0x08, 0x00, "set-policy-operation-v0")
	SetPolicyOperationV0Hint     = hint.MustHint(SetPolicyOperationV0Type, "0.0.1")
)

const PolicyOperationKey = "network_policy"

type PolicyOperationBodyV0 struct {
	Threshold                        Threshold     `json:"threshold"`
	TimeoutWaitingProposal           time.Duration `json:"timeout_waiting_proposal"`
	IntervalBroadcastingINITBallot   time.Duration `json:"interval_broadcasting_init_ballot"`
	IntervalBroadcastingProposal     time.Duration `json:"interval_broadcasting_proposal"`
	WaitBroadcastingACCEPTBallot     time.Duration `json:"wait_broadcasting_accept_ballot"`
	IntervalBroadcastingACCEPTBallot time.Duration `json:"interval_broadcasting_accept_ballot"`
	NumberOfActingSuffrageNodes      uint          `json:"number_of_acting_suffrage_nodes"`
	TimespanValidBallot              time.Duration `json:"timespan_valid_ballot"`
}

func (po PolicyOperationBodyV0) Bytes() []byte {
	return util.ConcatSlice([][]byte{
		po.Threshold.Bytes(),
		util.DurationToBytes(po.TimeoutWaitingProposal),
		util.DurationToBytes(po.IntervalBroadcastingINITBallot),
		util.DurationToBytes(po.IntervalBroadcastingProposal),
		util.DurationToBytes(po.WaitBroadcastingACCEPTBallot),
		util.DurationToBytes(po.IntervalBroadcastingACCEPTBallot),
		util.UintToBytes(po.NumberOfActingSuffrageNodes),
		util.DurationToBytes(po.TimespanValidBallot),
	})
}

func (po PolicyOperationBodyV0) Hash() valuehash.Hash {
	return valuehash.NewSHA256(po.Bytes())
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

	for k, d := range map[string]time.Duration{
		"TimeoutWaitingProposal":           spof.TimeoutWaitingProposal,
		"IntervalBroadcastingINITBallot":   spof.IntervalBroadcastingINITBallot,
		"IntervalBroadcastingProposal":     spof.IntervalBroadcastingProposal,
		"WaitBroadcastingACCEPTBallot":     spof.WaitBroadcastingACCEPTBallot,
		"IntervalBroadcastingACCEPTBallot": spof.IntervalBroadcastingACCEPTBallot,
		"TimespanValidBallot":              spof.TimespanValidBallot,
	} {
		if d < 0 {
			return xerrors.Errorf("%s is too narrow; duration=%v", k, d)
		}
	}

	if spof.NumberOfActingSuffrageNodes < 1 {
		return xerrors.Errorf("NumberOfActingSuffrageNodes must be over 0; %d", spof.NumberOfActingSuffrageNodes)
	}

	if err := spof.Threshold.IsValid(nil); err != nil {
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
	return util.ConcatSlice([][]byte{
		[]byte(spof.signer.String()),
		spof.token,
		spof.PolicyOperationBodyV0.Bytes(),
	})
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
	b []byte,
) (SetPolicyOperationV0, error) {
	if signer == nil {
		return SetPolicyOperationV0{}, xerrors.Errorf("empty privatekey")
	}

	fact := SetPolicyOperationFactV0{
		signer: signer.Publickey(),
		token:  token,
	}
	factHash := fact.Hash()
	var factSignature key.Signature
	if fs, err := signer.Sign(util.ConcatSlice([][]byte{factHash.Bytes(), b})); err != nil {
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

func (spo SetPolicyOperationV0) Fact() operation.Fact {
	return spo.SetPolicyOperationFactV0
}

func (spo SetPolicyOperationV0) Hash() valuehash.Hash {
	return spo.h
}

func (spo SetPolicyOperationV0) GenerateHash() (valuehash.Hash, error) {
	e := util.ConcatSlice([][]byte{
		spo.factHash.Bytes(),
		spo.factSignature.Bytes(),
	})

	return valuehash.NewSHA256(e), nil
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
	value := spo.SetPolicyOperationFactV0.PolicyOperationBodyV0

	var st state.StateUpdater
	if s, err := getState(PolicyOperationKey); err != nil {
		return nil, err
	} else if err := s.SetValue(value, value.Hash()); err != nil {
		return nil, err
	} else {
		st = s
	}

	return st, setState(st)
}
