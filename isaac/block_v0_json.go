package isaac

import (
	"encoding/json"

	"github.com/spikeekips/mitum/tree"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/encoder"
	"golang.org/x/xerrors"
)

type BlockV0PackJSON struct {
	encoder.JSONPackHintedHead
	MF ManifestV0           `json:"manifest"`
	CI BlockConsensusInfoV0 `json:"consensus"`
	OP *tree.AVLTree        `json:"operations"`
	ST *tree.AVLTree        `json:"states"`
}

func (bm BlockV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(BlockV0PackJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(bm.Hint()),
		MF:                 bm.ManifestV0,
		CI:                 bm.BlockConsensusInfoV0,
		OP:                 bm.operations,
		ST:                 bm.states,
	})
}

type BlockV0UnpackJSON struct {
	encoder.JSONPackHintedHead
	MF json.RawMessage `json:"manifest"`
	CI json.RawMessage `json:"consensus"`
	OP json.RawMessage `json:"operations"`
	ST json.RawMessage `json:"states"`
}

func (bm *BlockV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var nbm BlockV0UnpackJSON
	if err := enc.Unmarshal(b, &nbm); err != nil {
		return err
	}

	var mf ManifestV0
	if m, err := decodeManifest(enc, nbm.MF); err != nil {
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
	if tr, err := tree.DecodeAVLTree(enc, nbm.OP); err != nil {
		return err
	} else {
		operations = tr
	}

	if tr, err := tree.DecodeAVLTree(enc, nbm.ST); err != nil {
		return err
	} else {
		states = tr
	}

	bm.ManifestV0 = mf
	bm.BlockConsensusInfoV0 = ci
	bm.operations = &operations
	bm.states = &states

	return nil
}
