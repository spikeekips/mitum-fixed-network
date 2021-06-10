package base

import (
	"bytes"
	"encoding/base64"

	"golang.org/x/xerrors"
)

// NetworkID will be used to separate mitum network with the other network. For
// exampke, with different NetworkID, same Seal messsage will have different
// hash.
type NetworkID []byte

const MaxNetworkIDLength = 300

func (ni NetworkID) IsValid([]byte) error {
	if len(ni) < 1 {
		return xerrors.Errorf("empty NetworkID")
	} else if len(ni) > MaxNetworkIDLength {
		return xerrors.Errorf(
			"length of NetworkID too long; max=%d, but len=%d",
			MaxNetworkIDLength,
			len(ni),
		)
	}

	return nil
}

func (ni NetworkID) Equal(a NetworkID) bool {
	return bytes.Equal([]byte(ni), []byte(a))
}

func (ni NetworkID) MarshalText() ([]byte, error) {
	return []byte(base64.StdEncoding.EncodeToString(ni.Bytes())), nil
}

func (ni *NetworkID) UnmarshalText(b []byte) error {
	s, err := base64.StdEncoding.DecodeString(string(b))
	if err != nil {
		return err
	}

	*ni = NetworkID(s)

	return nil
}

func (ni NetworkID) Bytes() []byte {
	return []byte(ni)
}
