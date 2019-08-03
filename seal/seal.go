package seal

import (
	"encoding/json"
	"io"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/keypair"
)

var (
	SealHashHint string = "sl"
)

type Seal interface {
	rlp.Encoder
	common.IsValid
	Type() common.DataType
	Signer() keypair.PublicKey
	SignedAt() common.Time
	Signature() keypair.Signature
	Hash() hash.Hash
	Header() Header
	Body() Body
	Equal(Seal) bool
	CheckSignature([]byte) error
}

type Body interface {
	rlp.Encoder
	common.IsValid
	Type() common.DataType
	Hash() hash.Hash
}

type BaseSeal struct {
	t      common.DataType
	hash   hash.Hash
	header Header
	body   Body
}

func NewBaseSeal(body Body) BaseSeal {
	return BaseSeal{
		t: body.Type(),
		header: Header{
			bodyHash: body.Hash(),
		},
		body: body,
	}
}

func (bs BaseSeal) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":   bs.t,
		"hash":   bs.hash,
		"header": bs.header,
		"body":   bs.body,
	})
}

func (bs BaseSeal) EncodeRLP(w io.Writer) error {
	if bs.body == nil {
		return InvalidSealError.Newf("empty body")
	}

	return rlp.Encode(w, RLPEncodeSeal{
		Type:   bs.t,
		Hash:   bs.hash,
		Header: bs.header,
		Body:   bs.body,
	})
}

func (bs *BaseSeal) DecodeRLP(s *rlp.Stream) error {
	var d RLPDecodeSeal
	if err := s.Decode(&d); err != nil {
		return err
	}

	bs.t = d.Type
	bs.hash = d.Hash
	bs.header = d.Header
	//bs.body = d.Body

	return nil
}

func (bs BaseSeal) String() string {
	b, _ := json.Marshal(bs) // nolint
	return string(b)
}

func (bs BaseSeal) Type() common.DataType {
	return bs.t
}

func (bs *BaseSeal) SetType(t common.DataType) *BaseSeal {
	bs.t = t

	return bs
}

func (bs BaseSeal) Header() Header {
	return bs.header
}

func (bs *BaseSeal) SetHeader(header Header) *BaseSeal {
	bs.header = header
	return bs
}

func (bs BaseSeal) Body() Body {
	return bs.body
}

func (bs *BaseSeal) SetBody(body Body) *BaseSeal {
	bs.body = body
	return bs
}

func (bs BaseSeal) SignedAt() common.Time {
	return bs.header.signedAt
}

func (bs BaseSeal) Signer() keypair.PublicKey {
	return bs.header.signer
}

func (bs BaseSeal) Signature() keypair.Signature {
	return bs.header.signature
}

func (bs BaseSeal) Hash() hash.Hash {
	return bs.hash
}

func (bs *BaseSeal) SetHash(h hash.Hash) *BaseSeal {
	bs.hash = h
	return bs
}

func (bs BaseSeal) Equal(ns Seal) bool {
	if !bs.Hash().Equal(ns.Hash()) {
		return false
	}

	if !bs.Type().Equal(ns.Type()) {
		return false
	}

	if !bs.Body().Hash().Equal(ns.Body().Hash()) {
		return false
	}

	if !bs.Header().Equal(ns.Header()) {
		return false
	}

	return true
}

func (bs BaseSeal) makeHash() (hash.Hash, error) {
	b, err := rlp.EncodeToBytes(struct {
		Header Header
		Body   Body
	}{
		Header: bs.header,
		Body:   bs.body,
	})
	if err != nil {
		return hash.Hash{}, err
	}

	return hash.NewDoubleSHAHash(SealHashHint, b)
}

func (bs BaseSeal) BodyHash() hash.Hash {
	return bs.header.bodyHash
}

func (bs BaseSeal) IsValid() error {
	if bs.body == nil {
		return InvalidSealError.Newf("empty body")
	}

	if err := bs.hash.IsValid(); err != nil {
		return InvalidSealError.New(err)
	}

	if err := bs.header.IsValid(); err != nil {
		return err
	}

	if err := bs.body.IsValid(); err != nil {
		return InvalidSealError.New(err)
	}

	if !bs.header.bodyHash.Equal(bs.body.Hash()) {
		return InvalidSealError.Newf(
			"body.Hash() does not match; (bodyHash)%q != (body.Hash())%q",
			bs.header.bodyHash,
			bs.body.Hash(),
		)
	}

	return nil
}

func (bs *BaseSeal) Sign(pk keypair.PrivateKey, input []byte) error {
	var n []byte

	n = append(n, bs.body.Hash().Bytes()...)
	n = append(n, input...)

	sig, err := pk.Sign(n)
	if err != nil {
		return err
	}

	bs.header.bodyHash = bs.body.Hash()
	bs.header.signer = pk.PublicKey()
	bs.header.signature = sig
	bs.header.signedAt = common.Now()

	hash, err := bs.makeHash()
	if err != nil {
		return err
	}

	bs.hash = hash

	return nil
}

func (bs BaseSeal) CheckSignature(input []byte) error {
	var n []byte
	n = append(n, bs.body.Hash().Bytes()...)
	n = append(n, input...)

	return bs.header.signer.Verify(n, bs.header.signature)
}

type Header struct {
	bodyHash  hash.Hash
	signer    keypair.PublicKey
	signature keypair.Signature
	signedAt  common.Time
}

func (hd Header) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"signer":    hd.signer,
		"signature": hd.signature,
		"bodyHash":  hd.bodyHash,
		"signedAt":  hd.signedAt,
	})
}

func (hd Header) String() string {
	b, _ := json.Marshal(hd) // nolint
	return string(b)
}

func (hd Header) EncodeRLP(w io.Writer) error {
	if hd.signer == nil {
		return InvalidSealError.Newf("empty signer")
	}

	return rlp.Encode(w, struct {
		Signer    keypair.PublicKey
		Signature keypair.Signature
		BodyHash  hash.Hash
		SignedAt  common.Time
	}{
		Signer:    hd.signer,
		Signature: hd.signature,
		BodyHash:  hd.bodyHash,
		SignedAt:  hd.signedAt,
	})
}

func (hd *Header) DecodeRLP(s *rlp.Stream) error {
	var h struct {
		Signer    keypair.StellarPublicKey
		Signature keypair.Signature
		BodyHash  hash.Hash
		SignedAt  common.Time
	}

	if err := s.Decode(&h); err != nil {
		return err
	}

	hd.signer = h.Signer
	hd.signature = h.Signature
	hd.bodyHash = h.BodyHash
	hd.signedAt = h.SignedAt

	return nil
}

func (hd Header) IsValid() error {
	if hd.signer == nil {
		return InvalidSealError.Newf("empty signer")
	}

	if err := hd.signer.IsValid(); err != nil {
		return InvalidSealError.New(err)
	}

	if len(hd.signature) < 1 {
		return InvalidSealError.Newf("empty signature")
	}

	if err := hd.bodyHash.IsValid(); err != nil {
		return InvalidSealError.New(err)
	}

	if hd.signedAt.IsZero() {
		return InvalidSealError.Newf("zero signedAt time")
	}

	return nil
}

func (hd Header) Equal(nhd Header) bool {
	if !hd.bodyHash.Equal(nhd.bodyHash) {
		return false
	}

	if !hd.signer.Equal(nhd.signer) {
		return false
	}

	if !hd.signature.Equal(nhd.signature) {
		return false
	}

	if !hd.signedAt.Equal(nhd.signedAt) {
		return false
	}

	return true
}
