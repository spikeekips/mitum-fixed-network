package tree

import (
	"encoding/json"
	"sync"

	"github.com/spikeekips/avl"

	"github.com/spikeekips/mitum/base/valuehash"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
)

type AVLTreeJSONPacker struct {
	encoder.JSONPackHintedHead
	TY string         `json:"tree_type"`
	RT string         `json:"root_key"`
	RH valuehash.Hash `json:"root_hash"`
	NS []Node         `json:"nodes"`
}

func (at *AVLTree) MarshalJSON() ([]byte, error) {
	if at == nil || at.Empty() {
		return util.JSONMarshal(nil)
	}

	var nodes []Node
	if err := at.Traverse(func(node Node) (bool, error) {
		nodes = append(nodes, node.Immutable())

		return true, nil
	}); err != nil {
		return nil, err
	}

	var rh valuehash.Hash
	if h, err := at.RootHash(); err != nil {
		return nil, err
	} else {
		rh = h
	}

	return util.JSONMarshal(AVLTreeJSONPacker{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(at.Hint()),
		TY:                 "avl hashable tree",
		RT:                 string(at.Root().Key()),
		RH:                 rh,
		NS:                 nodes,
	})
}

type AVLTreeJSONUnpacker struct {
	RT string            `json:"root_key"`
	NS []json.RawMessage `json:"nodes"`
}

func (at *AVLTree) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var uat AVLTreeJSONUnpacker
	if err := enc.Unmarshal(b, &uat); err != nil {
		return err
	}

	var tr *avl.Tree
	np := avl.NewSyncMapNodePool(&sync.Map{})

	for _, r := range uat.NS {
		if n, err := DecodeNode(enc, r); err != nil {
			return err
		} else if err := np.Set(n); err != nil {
			return err
		}
	}

	if t, err := avl.NewTree([]byte(uat.RT), np); err != nil {
		return err
	} else {
		tr = t
	}

	at.Tree = tr

	return nil
}
