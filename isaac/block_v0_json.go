package isaac

import (
	"time"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/valuehash"
)

func (bm BlockV0) MarshalJSON() ([]byte, error) {
	return util.JSONMarshal(struct {
		H  valuehash.Hash `json:"hash"`
		HG Height         `json:"height"`
		RD Round          `json:"round"`
		PR valuehash.Hash `json:"proposal"`
		PB valuehash.Hash `json:"previous_block"`
		BO valuehash.Hash `json:"block_operations"`
		BS valuehash.Hash `json:"block_states"`
		IV VoteProof      `json:"init_voteproof"`
		AV VoteProof      `json:"accept_voteproof"`
		CA time.Time      `json:"created_at"`
	}{
		H:  bm.h,
		HG: bm.height,
		RD: bm.round,
		PR: bm.proposal,
		PB: bm.previousBlock,
		BO: bm.blockOperations,
		BS: bm.blockStates,
		IV: bm.initVoteProof,
		AV: bm.acceptVoteProof,
		CA: bm.createdAt,
	})
}
