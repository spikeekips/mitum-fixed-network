package state

import (
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/tree"
)

func (no *FixedTreeNode) UnmarshalBSON(b []byte) error {
	var ubno tree.BaseFixedTreeNode
	if err := bsonenc.Unmarshal(b, &ubno); err != nil {
		return err
	}

	no.BaseFixedTreeNode = ubno

	return nil
}
