// +build test

package operation

import (
	"encoding/json"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/key"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
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
	token []byte
	Key   string
	Value []byte
}

func (kvof KVOperationFact) IsValid([]byte) error {
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
		kvof.token,
		[]byte(kvof.Key),
		kvof.Value,
	)
}

func (kvof KVOperationFact) Token() []byte {
	return kvof.token
}

type KVOperation struct {
	KVOperationFact
	h  valuehash.Hash
	fs []FactSign
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
		token: token,
		Key:   k,
		Value: v,
	}
	var factSignature key.Signature
	if fs, err := signer.Sign(util.ConcatBytesSlice(fact.Hash().Bytes(), b)); err != nil {
		return KVOperation{}, err
	} else {
		factSignature = fs
	}

	kvo := KVOperation{
		KVOperationFact: fact,
		fs:              []FactSign{NewBaseFactSign(signer.Publickey(), factSignature)},
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
	bs := make([][]byte, len(kvo.fs))
	for i := range kvo.fs {
		bs[i] = kvo.fs[i].Bytes()
	}

	e := util.ConcatBytesSlice(kvo.Fact().Hash().Bytes(), util.ConcatBytesSlice(bs...))

	return valuehash.NewSHA256(e), nil
}

func (kvo KVOperation) Signs() []FactSign {
	return kvo.fs
}

func (kvo KVOperation) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(struct {
		jsonenc.HintedHead
		TK []byte         `json:"token"`
		K  string         `json:"key"`
		V  []byte         `json:"value"`
		H  valuehash.Hash `json:"hash"`
		FS []FactSign     `json:"fact_signs"`
	}{
		HintedHead: jsonenc.NewHintedHead(kvo.Hint()),
		TK:         kvo.token,
		K:          kvo.Key,
		V:          kvo.Value,
		H:          kvo.h,
		FS:         kvo.fs,
	})
}

func (kvo *KVOperation) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var ukvo struct {
		TK []byte            `json:"token"`
		K  string            `json:"key"`
		V  []byte            `json:"value"`
		H  valuehash.Bytes   `json:"hash"`
		FS []json.RawMessage `json:"fact_signs"`
	}

	if err := enc.Unmarshal(b, &ukvo); err != nil {
		return err
	}

	fs := make([][]byte, len(ukvo.FS))
	for i := range ukvo.FS {
		fs[i] = ukvo.FS[i]
	}

	return kvo.unpack(enc, ukvo.TK, ukvo.K, ukvo.V, ukvo.H, fs)
}

func (kvo KVOperation) MarshalBSON() ([]byte, error) {
	return bsonenc.Marshal(struct {
		HI hint.Hint      `bson:"_hint"`
		TK []byte         `bson:"token"`
		K  string         `bson:"key"`
		V  []byte         `bson:"value"`
		H  valuehash.Hash `bson:"hash"`
		FS []FactSign     `bson:"fact_signs"`
	}{
		HI: kvo.Hint(),
		TK: kvo.token,
		K:  kvo.Key,
		V:  kvo.Value,
		H:  kvo.h,
		FS: kvo.fs,
	})
}

func (kvo *KVOperation) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var ukvo struct {
		TK []byte          `bson:"token"`
		K  string          `bson:"key"`
		V  []byte          `bson:"value"`
		H  valuehash.Bytes `bson:"hash"`
		FS []bson.Raw      `bson:"fact_signs"`
	}

	if err := enc.Unmarshal(b, &ukvo); err != nil {
		return err
	}

	fs := make([][]byte, len(ukvo.FS))
	for i := range ukvo.FS {
		fs[i] = ukvo.FS[i]
	}

	return kvo.unpack(enc, ukvo.TK, ukvo.K, ukvo.V, ukvo.H, fs)
}

func (kvo *KVOperation) unpack(enc encoder.Encoder, tk []byte, k string, v []byte, h valuehash.Hash, bfs [][]byte) error {
	fs := make([]FactSign, len(bfs))
	for i := range bfs {
		if f, err := DecodeFactSign(enc, bfs[i]); err != nil {
			return err
		} else {
			fs[i] = f
		}
	}

	kvo.KVOperationFact = KVOperationFact{
		token: tk,
		Key:   k,
		Value: v,
	}

	kvo.h = h
	kvo.fs = fs

	return nil
}
