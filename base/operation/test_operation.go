// +build test

package operation

import (
	"encoding/json"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/valuehash"
)

var (
	MaxKeyKVOperation   int = 100
	MaxValueKVOperation int = 100
)

var (
	KVOperationFactType = hint.MustNewType(0xff, 0xf9, "kv-operation-fact")
	KVOperationFactHint = hint.MustHint(KVOperationFactType, "0.0.1")
	KVOperationType     = hint.MustNewType(0xff, 0xfa, "kv-operation")
	KVOperationHint     = hint.MustHint(KVOperationType, "0.0.1")
)

type KVOperationFact struct {
	signer key.Publickey
	token  []byte
	Key    string
	Value  []byte
}

func (kvof KVOperationFact) IsValid([]byte) error {
	if kvof.signer == nil {
		return xerrors.Errorf("fact has empty signer")
	}
	if err := kvof.Hint().IsValid(nil); err != nil {
		return err
	}

	if l := len(kvof.Key); l < 1 {
		return xerrors.Errorf("empty Key of KVOperation")
	} else if l > MaxKeyKVOperation {
		return xerrors.Errorf("Key of KVOperation over limit; %d > %d", l, MaxKeyKVOperation)
	}

	if kvof.Value != nil {
		if l := len(kvof.Value); l > MaxValueKVOperation {
			return xerrors.Errorf("Value of KVOperation over limit; %d > %d", l, MaxValueKVOperation)
		}
	}

	return nil
}

func (kvof KVOperationFact) Hint() hint.Hint {
	return KVOperationFactHint
}

func (kvof KVOperationFact) Hash() valuehash.Hash {
	return valuehash.NewSHA256(kvof.Bytes())
}

func (kvof KVOperationFact) Bytes() []byte {
	return util.ConcatBytesSlice(
		[]byte(kvof.signer.String()),
		kvof.token,
		[]byte(kvof.Key),
		kvof.Value,
	)
}

func (kvof KVOperationFact) Signer() key.Publickey {
	return kvof.signer
}

func (kvof KVOperationFact) Token() []byte {
	return kvof.token
}

type KVOperation struct {
	KVOperationFact
	h             valuehash.Hash
	factHash      valuehash.Hash
	factSignature key.Signature
}

func NewKVOperation(
	signer key.Privatekey,
	token []byte,
	k string,
	v []byte,
	b []byte,
) (KVOperation, error) {
	if signer == nil {
		return KVOperation{}, xerrors.Errorf("empty privatekey")
	}

	fact := KVOperationFact{
		signer: signer.Publickey(),
		token:  token,
		Key:    k,
		Value:  v,
	}
	factHash := fact.Hash()
	var factSignature key.Signature
	if fs, err := signer.Sign(util.ConcatBytesSlice(factHash.Bytes(), b)); err != nil {
		return KVOperation{}, err
	} else {
		factSignature = fs
	}

	kvo := KVOperation{
		KVOperationFact: fact,
		factHash:        factHash,
		factSignature:   factSignature,
	}

	if h, err := kvo.GenerateHash(); err != nil {
		return KVOperation{}, err
	} else {
		kvo.h = h
	}

	return kvo, nil
}

func (kvo KVOperation) IsValid(networkID []byte) error {
	return IsValidOperation(kvo, networkID)
}

func (kvo KVOperation) Hint() hint.Hint {
	return KVOperationHint
}

func (kvo KVOperation) Fact() base.Fact {
	return kvo.KVOperationFact
}

func (kvo KVOperation) Hash() valuehash.Hash {
	return kvo.h
}

func (kvo KVOperation) GenerateHash() (valuehash.Hash, error) {
	e := util.ConcatBytesSlice(kvo.factHash.Bytes(), kvo.factSignature.Bytes())

	return valuehash.NewSHA256(e), nil
}

func (kvo KVOperation) FactHash() valuehash.Hash {
	return kvo.factHash
}

func (kvo KVOperation) FactSignature() key.Signature {
	return kvo.factSignature
}

func (kvo KVOperation) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		jsonenc.HintedHead
		SG key.Publickey  `json:"signer"`
		TK []byte         `json:"token"`
		K  string         `json:"key"`
		V  []byte         `json:"value"`
		H  valuehash.Hash `json:"hash"`
		FH valuehash.Hash `json:"fact_hash"`
		FS key.Signature  `json:"fact_signature"`
	}{
		HintedHead: jsonenc.NewHintedHead(kvo.Hint()),
		SG:         kvo.signer,
		TK:         kvo.token,
		K:          kvo.Key,
		V:          kvo.Value,
		H:          kvo.h,
		FH:         kvo.factHash,
		FS:         kvo.factSignature,
	})
}

func (kvo *KVOperation) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ukvo struct {
		SG json.RawMessage `json:"signer"`
		TK []byte          `json:"token"`
		K  string          `json:"key"`
		V  []byte          `json:"value"`
		H  json.RawMessage `json:"hash"`
		FH json.RawMessage `json:"fact_hash"`
		FS key.Signature   `json:"fact_signature"`
	}

	if err := enc.Unmarshal(b, &ukvo); err != nil {
		return err
	}

	var err error

	var signer key.Publickey
	if signer, err = key.DecodePublickey(enc, ukvo.SG); err != nil {
		return err
	}

	var h, factHash valuehash.Hash
	if h, err = valuehash.Decode(enc, ukvo.H); err != nil {
		return err
	}
	if factHash, err = valuehash.Decode(enc, ukvo.FH); err != nil {
		return err
	}

	kvo.KVOperationFact = KVOperationFact{
		signer: signer,
		token:  ukvo.TK,
		Key:    ukvo.K,
		Value:  ukvo.V,
	}

	kvo.h = h
	kvo.factHash = factHash
	kvo.factSignature = ukvo.FS

	return nil
}

func (kvo KVOperation) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(struct {
		HI hint.Hint      `bson:"_hint"`
		SG key.Publickey  `bson:"signer"`
		TK []byte         `bson:"token"`
		K  string         `bson:"key"`
		V  []byte         `bson:"value"`
		H  valuehash.Hash `bson:"hash"`
		FH valuehash.Hash `bson:"fact_hash"`
		FS key.Signature  `bson:"fact_signature"`
	}{
		HI: kvo.Hint(),
		SG: kvo.signer,
		TK: kvo.token,
		K:  kvo.Key,
		V:  kvo.Value,
		H:  kvo.h,
		FH: kvo.factHash,
		FS: kvo.factSignature,
	})
}

func (kvo *KVOperation) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ukvo struct {
		SG bson.Raw        `bson:"signer"`
		TK []byte          `bson:"token"`
		K  string          `bson:"key"`
		V  []byte          `bson:"value"`
		H  valuehash.Bytes `bson:"hash"`
		FH valuehash.Bytes `bson:"fact_hash"`
		FS key.Signature   `bson:"fact_signature"`
	}

	if err := enc.Unmarshal(b, &ukvo); err != nil {
		return err
	}

	var err error

	var signer key.Publickey
	if signer, err = key.DecodePublickey(enc, ukvo.SG); err != nil {
		return err
	}

	kvo.KVOperationFact = KVOperationFact{
		signer: signer,
		token:  ukvo.TK,
		Key:    ukvo.K,
		Value:  ukvo.V,
	}

	kvo.h = ukvo.H
	kvo.factHash = ukvo.FH
	kvo.factSignature = ukvo.FS

	return nil
}
