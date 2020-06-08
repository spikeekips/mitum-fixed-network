package block

import (
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/tree"
	bsonenc "github.com/spikeekips/mitum/util/encoder/bson"
)

func (bm BlockV0) MarshalBSON() ([]byte, error) {
	m := bson.M{
		"manifest":  bm.ManifestV0,
		"consensus": bm.BlockConsensusInfoV0,
	}

	if bm.operations != nil && !bm.operations.Empty() {
		m["operations"] = bm.operations
	}

	if bm.states != nil && !bm.states.Empty() {
		m["states"] = bm.states
	}

	return bsonenc.Marshal(bsonenc.MergeBSONM(bsonenc.NewHintedDoc(bm.Hint()), m))
}

type BlockV0UnpackBSON struct {
	MF bson.Raw `bson:"manifest"`
	CI bson.Raw `bson:"consensus"`
	OP bson.Raw `bson:"operations,omitempty"`
	ST bson.Raw `bson:"states,omitempty"`
}

func (bm *BlockV0) UnpackBSON(b []byte, enc *bsonenc.Encoder) error {
	var nbm BlockV0UnpackBSON
	if err := enc.Unmarshal(b, &nbm); err != nil {
		return err
	}

	var mf ManifestV0
	if m, err := DecodeManifest(enc, nbm.MF); err != nil {
		return err
	} else if mv, ok := m.(ManifestV0); !ok {
		return xerrors.Errorf("not ManifestV0: type=%T", m)
	} else {
		mf = mv
	}

	var ci BlockConsensusInfoV0
	if m, err := decodeBlockConsensusInfo(enc, nbm.CI); err != nil {
		return err
	} else if mv, ok := m.(BlockConsensusInfoV0); !ok {
		return xerrors.Errorf("not ConsensusInfoV0: type=%T", m)
	} else {
		ci = mv
	}

	var operations, states tree.AVLTree
	if nbm.OP != nil {
		if tr, err := tree.DecodeAVLTree(enc, nbm.OP); err != nil {
			return err
		} else {
			operations = tr
		}
	}

	if nbm.ST != nil {
		if tr, err := tree.DecodeAVLTree(enc, nbm.ST); err != nil {
			return err
		} else {
			states = tr
		}
	}

	bm.ManifestV0 = mf
	bm.BlockConsensusInfoV0 = ci
	bm.operations = &operations
	bm.states = &states

	return nil
}
