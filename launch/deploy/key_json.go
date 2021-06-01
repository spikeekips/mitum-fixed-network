package deploy

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
)

type DeployKeyPackerJSON struct {
	K  string         `json:"key"`
	AA localtime.Time `json:"added_at"`
}

func (dk DeployKey) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(DeployKeyPackerJSON{
		K:  dk.k,
		AA: localtime.NewTime(dk.addedAt),
	})
}

type DeployKeyUnpackerJSON struct {
	K  string         `json:"key"`
	AA localtime.Time `json:"added_at"`
}

func (dk *DeployKey) UnmarshalJSON(b []byte) error {
	var udk DeployKeyUnpackerJSON
	if err := jsonenc.Unmarshal(b, &udk); err != nil {
		return err
	}

	return dk.unpack(udk.K, udk.AA.Time)
}
