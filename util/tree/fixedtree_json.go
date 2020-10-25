package tree

import (
	"github.com/btcsuite/btcutil/base58"

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
		s[i*3] = base58.Encode(key)
		s[i*3+1] = base58.Encode(h)
		s[i*3+2] = base58.Encode(v)

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
		ub[i] = base58.Decode(us.NS[i])
	}

	return ft.unpack(nil, ub)
}
