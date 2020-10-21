package block

import (
	"github.com/spikeekips/mitum/base"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/valuehash"
)

type ManifestV0PackJSON struct {
	jsonenc.HintedHead
	H  valuehash.Hash `json:"hash"`
	HT base.Height    `json:"height"`
	RD base.Round     `json:"round"`
	PR valuehash.Hash `json:"proposal"`
	PB valuehash.Hash `json:"previous_block"`
	BO valuehash.Hash `json:"block_operations"`
	BS valuehash.Hash `json:"block_states"`
	CF localtime.Time `json:"confirmed_at"`
	CA localtime.Time `json:"created_at"`
}

func (bm ManifestV0) MarshalJSON() ([]byte, error) {
	return jsonenc.Marshal(ManifestV0PackJSON{
		HintedHead: jsonenc.NewHintedHead(bm.Hint()),
		H:          bm.h,
		HT:         bm.height,
		RD:         bm.round,
		PR:         bm.proposal,
		PB:         bm.previousBlock,
		BO:         bm.operationsHash,
		BS:         bm.statesHash,
		CF:         localtime.NewTime(bm.confirmedAt),
		CA:         localtime.NewTime(bm.createdAt),
	})
}

type ManifestV0UnpackJSON struct {
	jsonenc.HintedHead
	H  valuehash.Bytes `json:"hash"`
	HT base.Height     `json:"height"`
	RD base.Round      `json:"round"`
	PR valuehash.Bytes `json:"proposal"`
	PB valuehash.Bytes `json:"previous_block"`
	BO valuehash.Bytes `json:"block_operations"`
	BS valuehash.Bytes `json:"block_states"`
	CF localtime.Time  `json:"confirmed_at"`
	CA localtime.Time  `json:"created_at"`
}

func (bm *ManifestV0) UnpackJSON(b []byte, enc *jsonenc.Encoder) error {
	var nbm ManifestV0UnpackJSON
	if err := enc.Unmarshal(b, &nbm); err != nil {
		return err
	}

	return bm.unpack(enc, nbm.H, nbm.HT, nbm.RD, nbm.PR, nbm.PB, nbm.BO, nbm.BS, nbm.CF.Time, nbm.CA.Time)
}
