package block

import (
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
	"github.com/spikeekips/mitum/util/tree"
)

func (bm BlockV0) MarshalBSON() ([]byte, error) {
	m := bson.M{
		"manifest":  bm.ManifestV0,
		"consensus": bm.ci,
	}

	if !bm.operationsTree.IsEmpty() {
		m["operations_tree"] = bm.operationsTree
	}

	if len(bm.operations) > 0 {
		m["operations"] = bm.operations
	}

	if !bm.statesTree.IsEmpty() {
		m["states_tree"] = bm.statesTree
	}

	if len(bm.states) > 0 {
		m["states"] = bm.states
	}

	return bsonenc.Marshal(bsonenc.MergeBSONM(bsonenc.NewHintedDoc(bm.Hint()), m))
}

type BlockV0UnpackBSON struct {
	MF  bson.Raw   `bson:"manifest"`
	CI  bson.Raw   `bson:"consensus"`
	OPT bson.Raw   `bson:"operations_tree,omitempty"`
	OP  []bson.Raw `bson:"operations,omitempty"`
	STT bson.Raw   `bson:"states_tree,omitempty"`
	ST  []bson.Raw `bson:"states,omitempty"`
}

func (bm *BlockV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var nbm BlockV0UnpackBSON
	if err := enc.Unmarshal(b, &nbm); err != nil {
		return err
	}

	if m, err := DecodeManifest(enc, nbm.MF); err != nil {
		return err
	} else if mv, ok := m.(ManifestV0); !ok {
		return xerrors.Errorf("not ManifestV0: type=%T", m)
	} else {
		bm.ManifestV0 = mv
	}

	if m, err := DecodeConsensusInfo(enc, nbm.CI); err != nil {
		return err
	} else if mv, ok := m.(ConsensusInfoV0); !ok {
		return xerrors.Errorf("not ConsensusInfoV0: type=%T", m)
	} else {
		bm.ci = mv
	}

	if nbm.OPT != nil {
		if tr, err := tree.DecodeFixedTree(enc, nbm.OPT); err != nil {
			return err
		} else {
			bm.operationsTree = tr
		}
	}

	ops := make([]operation.Operation, len(nbm.OP))
	for i := range nbm.OP {
		if op, err := operation.DecodeOperation(enc, nbm.OP[i]); err != nil {
			return err
		} else {
			ops[i] = op
		}
	}
	bm.operations = ops

	if nbm.STT != nil {
		if tr, err := tree.DecodeFixedTree(enc, nbm.STT); err != nil {
			return err
		} else {
			bm.statesTree = tr
		}
	}

	sts := make([]state.State, len(nbm.ST))
	for i := range nbm.ST {
		if st, err := state.DecodeState(enc, nbm.ST[i]); err != nil {
			return err
		} else {
			sts[i] = st
		}
	}
	bm.states = sts

	return nil
}
