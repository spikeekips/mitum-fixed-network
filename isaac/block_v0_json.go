package isaac

import (
	"encoding/json"

	"github.com/spikeekips/mitum/encoder"
	"github.com/spikeekips/mitum/localtime"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

type BlockV0PackJSON struct {
	encoder.JSONPackHintedHead
	H  valuehash.Hash     `json:"hash"`
	HT Height             `json:"height"`
	RD Round              `json:"round"`
	PR valuehash.Hash     `json:"proposal"`
	PB valuehash.Hash     `json:"previous_block"`
	BO valuehash.Hash     `json:"block_operations"`
	BS valuehash.Hash     `json:"block_states"`
	IV VoteProof          `json:"init_voteproof,omitempty"`
	AV VoteProof          `json:"accept_voteproof,omitempty"`
	CA localtime.JSONTime `json:"created_at"`
}

func (bm BlockV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(BlockV0PackJSON{
		JSONPackHintedHead: encoder.NewJSONPackHintedHead(bm.Hint()),
		H:                  bm.h,
		HT:                 bm.height,
		RD:                 bm.round,
		PR:                 bm.proposal,
		PB:                 bm.previousBlock,
		BO:                 bm.blockOperations,
		BS:                 bm.blockStates,
		IV:                 bm.initVoteProof,
		AV:                 bm.acceptVoteProof,
		CA:                 localtime.NewJSONTime(bm.createdAt),
	})
}

type BlockV0UnpackJSON struct {
	encoder.JSONPackHintedHead
	H  json.RawMessage    `json:"hash"`
	HT Height             `json:"height"`
	RD Round              `json:"round"`
	PR json.RawMessage    `json:"proposal"`
	PB json.RawMessage    `json:"previous_block"`
	BO json.RawMessage    `json:"block_operations"`
	BS json.RawMessage    `json:"block_states"`
	IV json.RawMessage    `json:"init_voteproof"`
	AV json.RawMessage    `json:"accept_voteproof"`
	CA localtime.JSONTime `json:"created_at"`
}

func (bm *BlockV0) UnpackJSON(b []byte, enc *encoder.JSONEncoder) error {
	var nbm BlockV0UnpackJSON
	if err := enc.Unmarshal(b, &nbm); err != nil {
		return err
	}

	var h, pr, pb, bo, bs valuehash.Hash
	var err error
	if h, err = decodeHash(enc, nbm.H); err != nil {
		return err
	}
	if pr, err = decodeHash(enc, nbm.PR); err != nil {
		return err
	}
	if pb, err = decodeHash(enc, nbm.PB); err != nil {
		return err
	}
	if bo, err = decodeHash(enc, nbm.BO); err != nil {
		return err
	}
	if bs, err = decodeHash(enc, nbm.BS); err != nil {
		return err
	}

	var iv, av VoteProof
	if nbm.IV != nil {
		if iv, err = decodeVoteProof(enc, nbm.IV); err != nil {
			return err
		}
	}
	if nbm.AV != nil {
		if av, err = decodeVoteProof(enc, nbm.AV); err != nil {
			return err
		}
	}

	bm.h = h
	bm.height = nbm.HT
	bm.round = nbm.RD
	bm.proposal = pr
	bm.previousBlock = pb
	bm.blockOperations = bo
	bm.blockStates = bs
	bm.initVoteProof = iv
	bm.acceptVoteProof = av
	bm.createdAt = nbm.CA.Time

	return nil
}
