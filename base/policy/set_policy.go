package policy

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
	"golang.org/x/xerrors"
)

var (
	SetPolicyFactV0Type = hint.MustNewType(0x01, 0x0c, "set-policy-operation-fact-v0")
	SetPolicyFactV0Hint = hint.MustHint(SetPolicyFactV0Type, "0.0.1")
	SetPolicyV0Type     = hint.MustNewType(0x01, 0x0d, "set-policy-operation-v0")
	SetPolicyV0Hint     = hint.MustHint(SetPolicyV0Type, "0.0.1")
)

type SetPolicyFactV0 struct {
	PolicyV0
	token []byte
}

func NewSetPolicyFactV0(po PolicyV0, token []byte) SetPolicyFactV0 {
	return SetPolicyFactV0{PolicyV0: po, token: token}
}

func (spof SetPolicyFactV0) Hint() hint.Hint {
	return SetPolicyFactV0Hint
}

func (spof SetPolicyFactV0) IsValid([]byte) error {
	return spof.PolicyV0.IsValid(nil)
}

func (spof SetPolicyFactV0) Bytes() []byte {
	return util.ConcatBytesSlice(spof.PolicyV0.Bytes(), spof.token)
}

func (spof SetPolicyFactV0) Hash() valuehash.Hash {
	return valuehash.NewSHA256(spof.Bytes())
}

func (spof SetPolicyFactV0) Token() []byte {
	return spof.token
}

type SetPolicyV0 struct {
	SetPolicyFactV0
	h  valuehash.Hash
	fs []operation.FactSign
}

func NewSetPolicyV0(
	po PolicyV0,
	token []byte,
	signer key.Privatekey,
	networkID base.NetworkID,
) (SetPolicyV0, error) {
	if signer == nil {
		return SetPolicyV0{}, xerrors.Errorf("empty privatekey")
	}

	fact := NewSetPolicyFactV0(po, token)
	if err := fact.IsValid(nil); err != nil {
		return SetPolicyV0{}, err
	}

	var factSig key.Signature
	if s, err := signer.Sign(util.ConcatBytesSlice(fact.Hash().Bytes(), networkID)); err != nil {
		return SetPolicyV0{}, err
	} else {
		factSig = s
	}

	spo := SetPolicyV0{
		SetPolicyFactV0: fact,
		fs:              []operation.FactSign{operation.NewBaseFactSign(signer.Publickey(), factSig)},
	}

	spo.h = spo.GenerateHash()

	return spo, nil
}

func (spo SetPolicyV0) Hint() hint.Hint {
	return SetPolicyV0Hint
}

func (spo SetPolicyV0) IsValid(networkID []byte) error {
	return operation.IsValidOperation(spo, networkID)
}

func (spo SetPolicyV0) Bytes() []byte {
	bs := make([][]byte, len(spo.fs)+1)
	bs[0] = spo.SetPolicyFactV0.Hash().Bytes()
	for i := range spo.fs {
		bs[i+1] = spo.fs[i].Bytes()
	}

	return util.ConcatBytesSlice(bs...)
}

func (spo SetPolicyV0) Hash() valuehash.Hash {
	return spo.h
}

func (spo SetPolicyV0) Fact() base.Fact {
	return spo.SetPolicyFactV0
}

func (spo SetPolicyV0) GenerateHash() valuehash.Hash {
	return valuehash.NewSHA256(spo.Bytes())
}

func (spo SetPolicyV0) Signs() []operation.FactSign {
	return spo.fs
}

func (spo SetPolicyV0) Process(
	getState func(key string) (state.State, bool, error),
	setState func(valuehash.Hash, ...state.State) error,
) error {
	var value state.HintedValue
	if v, err := state.NewHintedValue(spo.SetPolicyFactV0.PolicyV0); err != nil {
		return err
	} else {
		value = v
	}

	if s, found, err := getState(PolicyOperationKey); err != nil {
		return err
	} else if found {
		return xerrors.Errorf("already storeed; at this time, only new policy can be allowed")
	} else if ns, err := s.SetValue(value); err != nil {
		return err
	} else {
		return setState(spo.Hash(), ns)
	}
}
