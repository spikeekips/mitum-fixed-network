package common

import (
	"encoding/json"

	"github.com/stellar/go/keypair"
)

type Address string

func (a Address) IsValid() (keypair.KP, error) {
	return keypair.Parse(string(a))
}

func (a Address) Verify(input []byte, sig []byte) error {
	kp, err := a.IsValid()
	if err != nil {
		return err
	}

	return kp.Verify(input, sig)
}

func (a *Address) UnmarshalJSON(b []byte) error {
	var n string
	if err := json.Unmarshal(b, &n); err != nil {
		return err
	}

	if _, err := Address(n).IsValid(); err != nil {
		return err
	}

	*a = Address(n)
	return nil
}

type Seed struct {
	*keypair.Full
}

func RandomSeed() Seed {
	seed, _ := keypair.Random()
	return Seed{Full: seed}
}

func NewSeed(raw []byte) Seed {
	seed, _ := keypair.FromRawSeed([32]byte(RawHash(raw)))
	return Seed{Full: seed}
}

func (s Seed) Address() Address {
	return Address(s.Full.Address())
}
