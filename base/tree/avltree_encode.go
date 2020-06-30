package tree

import (
	"sync"

	"github.com/spikeekips/avl"

	"github.com/spikeekips/mitum/util/encoder"
)

func (at *AVLTree) unpack(enc encoder.Encoder, rootKey string, bNodes [][]byte) error {
	np := avl.NewSyncMapNodePool(&sync.Map{})

	for _, r := range bNodes {
		if n, err := DecodeNode(enc, r); err != nil {
			return err
		} else if err := np.Set(n); err != nil {
			return err
		}
	}

	var tr *avl.Tree
	if t, err := avl.NewTree([]byte(rootKey), np); err != nil {
		return err
	} else {
		tr = t
	}

	at.Tree = tr

	return nil
}
