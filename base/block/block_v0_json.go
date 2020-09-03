package block

import (
	"encoding/json"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/base/state"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/tree"
)

type BlockV0PackJSON struct {
	jsonenc.HintedHead
	MF  ManifestV0            `json:"manifest"`
	CI  ConsensusInfoV0       `json:"consensus"`
	OPT tree.FixedTree        `json:"operations_tree"`
	OP  []operation.Operation `json:"operations"`
	STT tree.FixedTree        `json:"states_tree"`
	ST  []state.State         `json:"states"`
}

func (bm BlockV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(BlockV0PackJSON{
		HintedHead: jsonenc.NewHintedHead(bm.Hint()),
		MF:         bm.ManifestV0,
		CI:         bm.ci,
		OPT:        bm.operationsTree,
		OP:         bm.operations,
		STT:        bm.statesTree,
		ST:         bm.states,
	})
}

type BlockV0UnpackJSON struct {
	jsonenc.HintedHead
	MF  json.RawMessage   `json:"manifest"`
	CI  json.RawMessage   `json:"consensus"`
	OPT json.RawMessage   `json:"operations_tree"`
	OP  []json.RawMessage `json:"operations"`
	STT json.RawMessage   `json:"states_tree"`
	ST  []json.RawMessage `json:"states"`
}

func (bm *BlockV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var nbm BlockV0UnpackJSON
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

	if m, err := decodeConsensusInfo(enc, nbm.CI); err != nil {
		return err
	} else if mv, ok := m.(ConsensusInfoV0); !ok {
		return xerrors.Errorf("not ConsensusInfoV0: type=%T", m)
	} else {
		bm.ci = mv
	}

	if tr, err := tree.DecodeFixedTree(enc, nbm.OPT); err != nil {
		return err
	} else {
		bm.operationsTree = tr
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

	if tr, err := tree.DecodeFixedTree(enc, nbm.STT); err != nil {
		return err
	} else {
		bm.statesTree = tr
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
