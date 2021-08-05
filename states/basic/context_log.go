package basicstates

import "github.com/rs/zerolog"

func (sctx StateSwitchContext) MarshalZerologObject(e *zerolog.Event) {
	var vid string
	if sctx.Voteproof() != nil {
		vid = sctx.Voteproof().ID()
	}

	e.
		Stringer("from", sctx.FromState()).
		Stringer("to", sctx.ToState()).
		Interface("error", sctx.Err()).
		Str("voteproof", vid)
}
