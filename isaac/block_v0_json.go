package isaac

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/util"
	"golang.org/x/xerrors"
)

type BlockV0PackJSON struct {
	encoder.JSONPackHintedHead
	MF BlockManifestV0      `json:"manifest"`
	CI BlockConsensusInfoV0 `json:"consensus"`
}

func (bm BlockV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(BlockV0PackJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(bm.Hint()),
		MF:                 bm.BlockManifestV0,
		CI:                 bm.BlockConsensusInfoV0,
	})
}

type BlockV0UnpackJSON struct {
	encoder.JSONPackHintedHead
	MF json.RawMessage `json:"manifest"`
	CI json.RawMessage `json:"consensus"`
}

func (bm *BlockV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var nbm BlockV0UnpackJSON
	if err := enc.Unmarshal(b, &nbm); err != nil {
		return err
	}

	var mf BlockManifestV0
	if m, err := decodeBlockManifest(enc, nbm.MF); err != nil {
		return err
	} else if mv, ok := m.(BlockManifestV0); !ok {
		return xerrors.Errorf("not BlockManifestV0: type=%T", m)
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

	bm.BlockManifestV0 = mf
	bm.BlockConsensusInfoV0 = ci

	return nil
}
