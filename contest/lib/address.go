package contestlib

import (
	"encoding/json"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base"
	bsonencoder "github.com/spikeekips/mitum/util/encoder/bson"
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
	"github.com/spikeekips/mitum/util/logging"
)

var (
	ContestAddressType = hint.MustNewType(0xd0, 0x00, "contest-address")
	ContestAddressHint = hint.MustHint(ContestAddressType, "0.0.1")
)

type ContestAddress string

func NewContestAddress(name string) (ContestAddress, error) {
	ca := ContestAddress(name)

	return ca, ca.IsValid(nil)
}

func (ca ContestAddress) String() string {
	return string(ca)
}

func (ca ContestAddress) Hint() hint.Hint {
	return ContestAddressHint
}

func (ca ContestAddress) IsValid([]byte) error {
	if s := strings.TrimSpace(ca.String()); len(s) < 1 {
		return xerrors.Errorf("empty address")
	}

	return nil
}

func (ca ContestAddress) Equal(a base.Address) bool {
	if ca.Hint().Type() != a.Hint().Type() {
		return false
	}

	return ca == a.(ContestAddress)
}

func (ca ContestAddress) Bytes() []byte {
	return []byte(ca)
}

func (ca ContestAddress) MarshalJSON() ([]byte, error) {
	return jsonencoder.Marshal(struct {
		jsonencoder.HintedHead
		A string `json:"address"`
	}{
		HintedHead: jsonencoder.NewHintedHead(ca.Hint()),
		A:          ca.String(),
	})
}

func (ca *ContestAddress) UnpackJSON(b []byte, _ *jsonencoder.Encoder) error {
	var s struct {
		jsonencoder.HintedHead
		A string `json:"address"`
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	} else if len(s.A) < 1 {
		return xerrors.Errorf("not enough address")
	}

	*ca = ContestAddress(s.A)

	return nil
}

func (ca ContestAddress) MarshalBSON() ([]byte, error) {
	return bsonencoder.Marshal(bsonencoder.MergeBSONM(
		bsonencoder.NewHintedDoc(ca.Hint()),
		bson.M{"address": ca.String()},
	))
}

func (ca *ContestAddress) UnpackBSON(b []byte, enc *bsonencoder.Encoder) error {
	var s struct {
		A string `bson:"address"`
	}
	if err := enc.Unmarshal(b, &s); err != nil {
		return err
	} else if len(s.A) < 1 {
		return xerrors.Errorf("not enough address")
	}

	*ca = ContestAddress(s.A)

	return nil
}

func (ca ContestAddress) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	if !verbose {
		return e.Str(key, ca.String())
	}

	return e.Dict(key, logging.Dict().
		Str("address", ca.String()).
		HintedVerbose("hint", ca.Hint(), true),
	)
}
