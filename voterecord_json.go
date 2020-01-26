package mitum

import (
	"encoding/json"

	"github.com/spikeekips/mitum/localtime"
)

func (vrc VoteRecordINIT) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"node":          vrc.node,
		"voted_at":      localtime.NewJSONTime(vrc.votedAt),
		"previousBlock": vrc.previousBlock.String(),
		"previousRound": vrc.previousRound,
	})
}

func (vrc VoteRecordSIGN) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"node":      vrc.node,
		"voted_at":  localtime.NewJSONTime(vrc.votedAt),
		"proposal":  vrc.proposal.String(),
		"new_block": vrc.newBlock.String(),
	})
}

func (vrc VoteRecordACCEPT) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"node":      vrc.node,
		"voted_at":  localtime.NewJSONTime(vrc.votedAt),
		"proposal":  vrc.proposal.String(),
		"new_block": vrc.newBlock.String(),
	})
}
