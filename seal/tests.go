// +build test

package seal

import (
	"crypto/rand"
	"io"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/keypair"
	"golang.org/x/xerrors"
)

func init() {
	common.SetTestLogger(log)
}

func NewRandomSealHash() hash.Hash {
	b := make([]byte, 4)
	_, _ = rand.Read(b)

	h, _ := hash.NewHash(SealHashHint, b)
	return h
}

type SealBodyTest struct {
	hash hash.Hash
	T    common.DataType
	A    string
	B    uint64
}

func (t SealBodyTest) Type() common.DataType {
	return t.T
}

func (t SealBodyTest) makeHash() (hash.Hash, error) {
	b, err := rlp.EncodeToBytes(t)
	if err != nil {
		log.Error("Hash() failed", "error", err)
		return hash.Hash{}, err
	}

	h, err := hash.NewDoubleSHAHash("ts", b)
	if err != nil {
		log.Error("Hash() failed", "error", err)
		return hash.Hash{}, err
	}

	return h, nil
}

func (t SealBodyTest) Hash() hash.Hash {
	return t.hash
}

func (t SealBodyTest) IsValid() error {
	if t.T.Empty() {
		return xerrors.Errorf("empty T")
	}

	if len(t.A) < 1 {
		return xerrors.Errorf("empty A")
	}

	if t.B < 1 {
		return xerrors.Errorf("negative B")
	}

	return nil
}

func (t SealBodyTest) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, struct {
		H hash.Hash
		T common.DataType
		A string
		B uint64
	}{
		H: t.hash,
		T: t.T,
		A: t.A,
		B: t.B,
	})
}

func (t *SealBodyTest) DecodeRLP(s *rlp.Stream) error {
	var h struct {
		H hash.Hash
		T common.DataType
		A string
		B uint64
	}

	if err := s.Decode(&h); err != nil {
		return err
	}

	t.hash = h.H
	t.T = h.T
	t.A = h.A
	t.B = h.B

	return nil
}

func NewSealBody(a string, b uint64) SealBodyTest {
	dt := common.NewDataType(33, "test-seal-body")
	ts := SealBodyTest{T: dt, A: a, B: b}
	ts.hash, _ = ts.makeHash()
	return ts
}

func NewSealBodySigned(pk keypair.PrivateKey, a string, b uint64) (Seal, error) {
	body := NewSealBody(a, b)
	sl := NewBaseSeal(body)
	if err := sl.Sign(pk, []byte{}); err != nil {
		return nil, err
	}

	return sl, nil
}
