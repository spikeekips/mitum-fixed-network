package base

import (
	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/logging"
)

func (vp VoteproofV0) MarshalLog(key string, e logging.Emitter, verbose bool) logging.Emitter {
	if !verbose {
		ev := logging.Dict().
			Hinted("height", vp.height).
			Hinted("round", vp.round).
			Hinted("stage", vp.stage).
			Bool("is_closed", vp.closed).
			Str("result", vp.result.String()).
			Int("number_of_votes", len(vp.votes)).
			Int("number_of_ballots", len(vp.ballots))

		if vp.IsFinished() {
			ev = ev.Hinted("fact", vp.majority.Hash()).
				Time("finished_at", vp.finishedAt)
		}

		return e.Dict(key, ev)
	}

	r, _ := jsonencoder.Marshal(vp)

	return e.RawJSON(key, r)
}
