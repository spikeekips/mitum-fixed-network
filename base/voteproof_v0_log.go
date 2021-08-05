package base

import (
	"github.com/rs/zerolog"
)

func (vp VoteproofV0) MarshalZerologObject(e *zerolog.Event) {
	e.
		Str("id", vp.ID()).
		Int64("height", vp.height.Int64()).
		Uint64("round", vp.round.Uint64()).
		Stringer("stage", vp.stage).
		Bool("is_closed", vp.closed).
		Stringer("result", vp.result).
		Int("number_of_votes", len(vp.votes))

	if vp.IsFinished() {
		if vp.majority != nil {
			e.Stringer("fact", vp.majority.Hash())
		}

		e.Time("finished_at", vp.finishedAt)
	}
}
