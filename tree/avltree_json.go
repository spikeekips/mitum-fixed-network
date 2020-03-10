package tree

import (
	"encoding/json"
	"sync"

	"github.com/spikeekips/avl"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/util"
)

type AVLTreeJSONPacker struct {
	encoder.JSONPackHintedHead
	TY string `json:"tree_type"`
	RT string `json:"root_key"`
	RH []byte `json:"root_hash"`
	NS []Node `json:"nodes"`
}

func (at AVLTree) MarshalJSON() ([]byte, error) {
	var nodes []Node
	if err := at.Traverse(func(node Node) (bool, error) {
		nodes = append(nodes, node)

		return true, nil
	}); err != nil {
		return nil, err
	}

	return util.JSONMarshal(AVLTreeJSONPacker{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(at.Hint()),
		TY:                 "avl hashable tree",
		RT:                 string(at.Root().Key()),
		RH:                 at.Root().Hash(),
		NS:                 nodes,
	})
}

type AVLTreeJSONUnpacker struct {
	RT string            `json:"root_key"`
	RH []byte            `json:"root_hash"`
	NS []json.RawMessage `json:"nodes"`
}

func (at *AVLTree) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var uat AVLTreeJSONUnpacker
	if err := enc.Unmarshal(b, &uat); err != nil {
		return err
	}

	np := avl.NewSyncMapNodePool(&sync.Map{})

	if tree, err := avl.NewTree([]byte(uat.RT), np); err != nil {
		return err
	} else {
		at.Tree = tree
	}

	return nil
}
