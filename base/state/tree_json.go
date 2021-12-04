package state

import (
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/tree"
)

func (no *FixedTreeNode) UnmarshalJSON(b []byte) error {
	var ubno tree.BaseFixedTreeNode
	if err := jsonenc.Unmarshal(b, &ubno); err != nil {
		return err
	}

	no.BaseFixedTreeNode = ubno

	return nil
}
