package isaac

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type BlockManifestV0PackJSON struct {
	encoder.JSONPackHintedHead
	H  valuehash.Hash     `json:"hash"`
	HT Height             `json:"height"`
	RD Round              `json:"round"`
	PR valuehash.Hash     `json:"proposal"`
	PB valuehash.Hash     `json:"previous_block"`
	BO valuehash.Hash     `json:"block_operations"`
	BS valuehash.Hash     `json:"block_states"`
	CA localtime.JSONTime `json:"created_at"`
}

func (bm BlockManifestV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(BlockManifestV0PackJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(bm.Hint()),
		H:                  bm.h,
		HT:                 bm.height,
		RD:                 bm.round,
		PR:                 bm.proposal,
		PB:                 bm.previousBlock,
		BO:                 bm.operationsHash,
		BS:                 bm.statesHash,
		CA:                 localtime.NewJSONTime(bm.createdAt),
	})
}

type BlockManifestV0UnpackJSON struct {
	encoder.JSONPackHintedHead
	H  json.RawMessage    `json:"hash"`
	HT Height             `json:"height"`
	RD Round              `json:"round"`
	PR json.RawMessage    `json:"proposal"`
	PB json.RawMessage    `json:"previous_block"`
	BO json.RawMessage    `json:"block_operations"`
	BS json.RawMessage    `json:"block_states"`
	CA localtime.JSONTime `json:"created_at"`
}

func (bm *BlockManifestV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var nbm BlockManifestV0UnpackJSON
	if err := enc.Unmarshal(b, &nbm); err != nil {
		return err
	}

	var h, pr, pb, bo, bs valuehash.Hash
	var err error
	if h, err = valuehash.Decode(enc, nbm.H); err != nil {
		return err
	}
	if pr, err = valuehash.Decode(enc, nbm.PR); err != nil {
		return err
	}
	if pb, err = valuehash.Decode(enc, nbm.PB); err != nil {
		return err
	}
	if bo, err = valuehash.Decode(enc, nbm.BO); err != nil {
		return err
	}
	if bs, err = valuehash.Decode(enc, nbm.BS); err != nil {
		return err
	}

	bm.h = h
	bm.height = nbm.HT
	bm.round = nbm.RD
	bm.proposal = pr
	bm.previousBlock = pb
	bm.operationsHash = bo
	bm.statesHash = bs
	bm.createdAt = nbm.CA.Time

	return nil
}
