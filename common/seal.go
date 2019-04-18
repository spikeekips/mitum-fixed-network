package common

import (
	"encoding"
	"encoding/base64"
	"encoding/json"

	"github.com/Masterminds/semver"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	CurrentSealVersion semver.Version = *semver.MustParse("0.1.0-proto")
)

type SealType uint

const (
	_ SealType = iota
	SealedSeal
	BallotSeal
	TransactionSeal
)

func (s SealType) String() string {
	switch s {
	case SealedSeal:
		return "sealed"
	case BallotSeal:
		return "ballot"
	case TransactionSeal:
		return "transaction"
	default:
		return ""
	}
}

func (s SealType) MarshalText() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *SealType) UnmarshalText(b []byte) error {
	var i string
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}

	switch i {
	case "sealed":
		*s = SealedSeal
	case "ballot":
		*s = BallotSeal
	case "transaction":
		*s = TransactionSeal
	default:
		return UnknownSealTypeError
	}

	return nil
}

type Seal struct {
	Version   semver.Version
	Type      SealType
	Source    Address
	Signature Signature
	hash      Hash
	Body      []byte
}

func NewSeal(t SealType, body Hashable) (Seal, error) {
	hash, encoded, err := body.Hash()
	if err != nil {
		return Seal{}, err
	}

	return Seal{
		Type:    t,
		Version: CurrentSealVersion,
		hash:    hash,
		Body:    encoded,
	}, nil
}

func (s Seal) MarshalBinary() ([]byte, error) {
	var err error

	version, err := json.Marshal(&s.Version)
	if err != nil {
		return nil, err
	}

	hash, err := json.Marshal(s.hash)
	if err != nil {
		return nil, err
	}

	return Encode([]interface{}{
		version,
		s.Type,
		s.Source,
		s.Signature,
		hash,
		s.Body,
	})
}

func (s *Seal) UnmarshalBinary(b []byte) error {
	var m []rlp.RawValue
	if err := Decode(b, &m); err != nil {
		return err
	}

	var version *semver.Version
	{
		var vs []byte
		if err := Decode(m[0], &vs); err != nil {
			return err
		} else if err := json.Unmarshal(vs, &version); err != nil {
			return err
		}
	}

	var sealType SealType
	{
		if err := Decode(m[1], &sealType); err != nil {
			return err
		}
	}

	var source Address
	{
		if err := Decode(m[2], &source); err != nil {
			return err
		}
	}

	var signature Signature
	{
		if err := Decode(m[3], &signature); err != nil {
			return err
		}
	}

	var hash Hash
	{
		var vs []byte
		if err := Decode(m[4], &vs); err != nil {
			return err
		} else if err := json.Unmarshal(vs, &hash); err != nil {
			return err
		}
	}

	var body []byte
	if err := Decode(m[5], &body); err != nil {
		return err
	}

	s.Version = *version
	s.hash = hash
	s.Type = sealType
	s.Signature = signature
	s.Source = source
	s.Body = body

	return nil
}

func (s Seal) Hash() (Hash, []byte, error) {
	encoded, err := s.MarshalBinary()
	if err != nil {
		return Hash{}, nil, err
	}

	return NewHash("sl", encoded), encoded, nil
}

func (s *Seal) Sign(networkID NetworkID, seed Seed) error {
	signature, err := NewSignature(networkID, seed, s.hash)
	if err != nil {
		return err
	}

	s.Source = seed.Address()
	s.Signature = signature
	return nil
}

func (s Seal) CheckSignature(networkID NetworkID) error {
	err := s.Source.Verify(
		append(networkID, s.hash.Bytes()...),
		[]byte(s.Signature),
	)
	if err != nil {
		return err
	}

	return nil
}

func (s Seal) MarshalText() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"version":   &s.Version,
		"type":      s.Type,
		"source":    s.Source,
		"signature": s.Signature,
		"hash":      s.hash,
		"body":      base64.StdEncoding.EncodeToString(s.Body),
	})
}

func (s *Seal) UnmarshalText(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}

	var version semver.Version
	if err := json.Unmarshal(raw["version"], &version); err != nil {
		return err
	}

	var source Address
	if err := json.Unmarshal(raw["source"], &source); err != nil {
		return err
	}

	var sealType SealType
	if err := json.Unmarshal(raw["type"], &sealType); err != nil {
		return err
	}

	var signature Signature
	if err := json.Unmarshal(raw["signature"], &signature); err != nil {
		return err
	}

	var hash Hash
	if err := json.Unmarshal(raw["hash"], &hash); err != nil {
		return err
	}

	var body []byte
	{
		var c string
		if err := json.Unmarshal(raw["body"], &c); err != nil {
			return err
		} else if d, err := base64.StdEncoding.DecodeString(c); err != nil {
			return err
		} else {
			body = d
		}
	}

	s.Version = version
	s.Type = sealType
	s.Source = source
	s.Signature = signature
	s.hash = hash
	s.Body = body

	return nil
}

func (s Seal) UnmarshalBody(i encoding.BinaryUnmarshaler) error {
	return i.UnmarshalBinary(s.Body)
}

func (s Seal) String() string {
	b, _ := json.MarshalIndent(s, "", "  ")
	return string(b)
}
