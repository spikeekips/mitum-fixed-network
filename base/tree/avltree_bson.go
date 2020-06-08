package tree

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/base/valuehash"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (at *AVLTree) MarshalBSON() ([]byte, error) {
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

	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(at.Hint()),
		bson.M{
			"tree_type": "avl hashable tree",
			"root_key":  string(at.Root().Key()),
			"root_hash": rh,
			"nodes":     nodes,
		},
	))
}

type AVLTreeBSONUnpacker struct {
	RT string     `bson:"root_key"`
	NS []bson.Raw `bson:"nodes"`
}

func (at *AVLTree) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var uat AVLTreeBSONUnpacker
	if err := enc.Unmarshal(b, &uat); err != nil {
		return err
	}

	ns := make([][]byte, len(uat.NS))
	for i, b := range uat.NS {
		ns[i] = b
	}

	return at.unpack(enc, uat.RT, ns)
}
