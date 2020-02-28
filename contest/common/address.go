package common

import (
	"encoding/json"
	"fmt"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/hint"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util"
)

var ContestAddressHint hint.Hint = hint.MustHint(hint.Type{0xd0, 0x00}, "0.1")

type ContestAddress string

func NewContestAddress(id int) ContestAddress {
	return ContestAddress(fmt.Sprintf("node:%02d", id))
}

func (sa ContestAddress) String() string {
	return string(sa)
}

func (sa ContestAddress) Hint() hint.Hint {
	return ContestAddressHint
}

func (sa ContestAddress) IsValid([]byte) error {
	if len(sa) < 1 {
		return xerrors.Errorf("empty address")
	}

	return nil
}

func (sa ContestAddress) Equal(a isaac.Address) bool {
	if sa.Hint().Type() != a.Hint().Type() {
		return false
	}

	return sa == a.(ContestAddress)
}

func (sa ContestAddress) Bytes() []byte {
	return []byte(sa)
}

func (sa ContestAddress) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		encoder.JSONPackHintedHead
		A string `json:"address"`
	}{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(sa.Hint()),
		A:                  sa.String(),
	})
}

func (sa *ContestAddress) UnpackJSON(b []byte, _ *encoder.JSONEncoder) error {
	var s struct {
		encoder.JSONPackHintedHead
		A string `json:"address"`
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	} else if err := sa.Hint().IsCompatible(s.H); err != nil {
		return err
	} else if len(s.A) < 5 {
		return xerrors.Errorf("not enough address")
	}

	*sa = ContestAddress(s.A)

	return nil
}
