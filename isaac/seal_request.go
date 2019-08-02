package isaac

import (
	"encoding/json"
	"io"
	"reflect"

	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/seal"
)

var (
	RquestType     common.DataType = common.NewDataType(4, "request")
	RquestHashHint string          = "request"
)

type RequestKind uint

const (
	RequestUnknown RequestKind = iota
	RequestVoteProof
)

func (rs RequestKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(rs.String())
}

func (rs RequestKind) IsValid() error {
	switch rs {
	case RequestVoteProof:
		return nil
	default:
		return xerrors.Errorf("unknown request; %q", rs)
	}
}

func (rs RequestKind) String() string {
	switch rs {
	case RequestVoteProof:
		return "vote-proof-request"
	default:
		return ""
	}
}

type Request struct {
	seal.BaseSeal
	body RequestBody
}

func NewRequest(request RequestKind, params ...interface{}) (seal.Seal, error) {
	m, err := common.SetStringMap(params...)
	if err != nil {
		return nil, err
	}

	body := RequestBody{request: request, params: m}
	h, err := body.makeHash()
	if err != nil {
		return nil, err
	}
	body.hash = h

	return Request{
		BaseSeal: seal.NewBaseSeal(body),
		body:     body,
	}, nil
}

func (rs Request) Request() RequestKind {
	return rs.body.request
}

func (rs Request) Params() map[string]interface{} {
	return rs.body.params
}

func (rs Request) Has(key string) bool {
	_, found := rs.body.params[key]
	return found
}

func (rs Request) Get(key string, v interface{}) error {
	p, found := rs.body.params[key]
	if !found {
		return xerrors.Errorf("param not found; key=%q", key)
	}

	reflect.ValueOf(p).Elem().Set(reflect.ValueOf(v))

	return nil
}

type RequestBody struct {
	hash    hash.Hash
	request RequestKind
	params  map[string]interface{}
}

func (rb RequestBody) Hash() hash.Hash {
	return rb.hash
}

func (rb RequestBody) Type() common.DataType {
	return RquestType
}

func (rb RequestBody) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"hash":    rb.hash,
		"request": rb.request,
		"params":  rb.params,
	})
}

func (rb RequestBody) String() string {
	b, _ := common.EncodeJSON(rb, true, false)
	return string(b)
}

func (rb RequestBody) EncodeRLP(w io.Writer) error {
	var params []interface{}
	for k, v := range rb.params {
		params = append(params, k)
		params = append(params, v)
	}

	return rlp.Encode(w, struct {
		H hash.Hash
		R RequestKind
		P []interface{}
	}{
		H: rb.hash,
		R: rb.request,
		P: params,
	})
}

func (rb *RequestBody) DecodeRLP(s *rlp.Stream) error {
	var body struct {
		H hash.Hash
		R RequestKind
		P []interface{}
	}
	if err := s.Decode(&body); err != nil {
		return err
	}

	rb.hash = body.H
	rb.request = body.R

	if len(body.P)%2 != 0 {
		return xerrors.Errorf("slice of params should be paired")
	}

	params, err := common.SetStringMap(body.P)
	if err != nil {
		return err
	}

	rb.params = params

	return nil
}

func (rb RequestBody) makeHash() (hash.Hash, error) {
	var params []interface{}
	for k, v := range rb.params {
		params = append(params, k)
		params = append(params, v)
	}

	b, err := rlp.EncodeToBytes([]interface{}{
		rb.request,
		params,
	})
	if err != nil {
		return hash.Hash{}, err
	}
	h, err := hash.NewDoubleSHAHash(RquestHashHint, b)
	if err != nil {
		return hash.Hash{}, err
	}

	return h, nil
}

func (rb RequestBody) IsValid() error {
	if err := rb.request.IsValid(); err != nil {
		return err
	}

	return nil
}
