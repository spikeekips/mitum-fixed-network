package memberlist

import (
	"github.com/spikeekips/mitum/network"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type ConnInfoPackerJSON struct {
	A string `json:"address"`
}

func (ci ConnInfo) MarshalJSON() ([]byte, error) {
	h := network.HTTPConnInfoPackerJSON{
		HintedHead: jsonenc.NewHintedHead(ci.HTTPConnInfo.Hint()),
		U:          ci.URL().String(),
		I:          ci.Insecure(),
	}

	i := ConnInfoPackerJSON{
		A: ci.Address,
	}

	return jsonenc.Marshal(struct {
		*network.HTTPConnInfoPackerJSON
		*ConnInfoPackerJSON
	}{
		HTTPConnInfoPackerJSON: &h,
		ConnInfoPackerJSON:     &i,
	})
}

func (ci *ConnInfo) UnmarshalJSON(b []byte) error {
	var uhc network.HTTPConnInfo
	if err := jsonenc.Unmarshal(b, &uhc); err != nil {
		return err
	}

	var uc ConnInfoPackerJSON
	if err := jsonenc.Unmarshal(b, &uc); err != nil {
		return err
	}

	ci.HTTPConnInfo = uhc
	ci.Address = uc.A

	return nil
}
