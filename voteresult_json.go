package mitum

import "encoding/json"

func (vr VoteResult) MarshalJSON() ([]byte, error) {
	votes := map[string]interface{}{}
	for k, v := range vr.votes {
		votes[k.String()] = v
	}

	return json.Marshal(map[string]interface{}{
		"height": vr.height,
		"round":  vr.round,
		"stage":  vr.stage,
		"result": vr.result,
		//"majority": vr.majority,
		//"votes":    votes,
	})
}
