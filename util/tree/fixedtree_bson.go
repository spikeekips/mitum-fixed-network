package tree

import (
	"go.mongodb.org/mongo-driver/bson"

	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (ft FixedTree) MarshalBSON() ([]byte, error) {
	nodes := make([][]byte, ft.Len()*3)
	if err := ft.Traverse(func(i int, key, h, v []byte) (bool, error) {
		nodes[i*3] = key
		nodes[i*3+1] = h
		nodes[i*3+2] = v

		return true, nil
	}); err != nil {
		return nil, err
	}

	return bsonenc.Marshal(bsonenc.MergeBSONM(
		bsonenc.NewHintedDoc(ft.Hint()),
		bson.M{
			"nodes": nodes,
		},
	))
}

type FixedTreeBSONUnpacker struct {
	NS []bson.Raw `bson:"nodes"`
}

func (ft *FixedTree) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var us FixedTreeBSONUnpacker
	if err := enc.Unmarshal(b, &us); err != nil {
		return err
	}

	bs := make([][]byte, len(us.NS))
	for i := range us.NS {
		if len(us.NS[i]) < 1 {
			continue
		}

		bs[i] = us.NS[i]
	}

	return ft.unpack(nil, bs)
}
