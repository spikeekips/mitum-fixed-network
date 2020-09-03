package tree

import (
	"encoding/hex"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type FixedTreeJSONPacker struct {
	jsonenc.HintedHead
	NS []string `json:"nodes"`
}

func (ft FixedTree) MarshalJSON() ([]byte, error) {
	if ft.IsEmpty() {
		return jsonenc.Marshal(nil)
	}

	s := make([]string, len(ft.nodes))
	if err := ft.Traverse(func(i int, key, h, v []byte) (bool, error) {
		s[i*3] = hex.EncodeToString(key)
		s[i*3+1] = hex.EncodeToString(h)
		s[i*3+2] = hex.EncodeToString(v)

		return true, nil
	}); err != nil {
		return nil, err
	}

	return jsonenc.Marshal(FixedTreeJSONPacker{
		HintedHead: jsonenc.NewHintedHead(ft.Hint()),
		NS:         s,
	})
}

type FixedTreeJSONUnpacker struct {
	NS []string `json:"nodes"`
}

func (ft *FixedTree) UnmarshalJSON(b []byte) error {
	var us FixedTreeJSONUnpacker
	if err := jsonenc.Unmarshal(b, &us); err != nil {
		return err
	}

	ub := make([][]byte, len(us.NS))
	for i := range us.NS {
		if b, err := hex.DecodeString(us.NS[i]); err != nil {
			return err
		} else {
			ub[i] = b
		}
	}

	return ft.unpack(nil, ub)
}
