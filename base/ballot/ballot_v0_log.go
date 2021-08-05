package ballot

import (
	"github.com/rs/zerolog"
)

func marshalLog(ballot Ballot, e *zerolog.Event) {
	e.
		Stringer("hash", ballot.Hash()).
		Stringer("stage", ballot.Stage()).
		Stringer("node", ballot.Node()).
		Int64("height", ballot.Height().Int64()).
		Uint64("round", ballot.Round().Uint64())
}

func (ib INITV0) MarshalZerologObject(e *zerolog.Event) {
	marshalLog(ib, e)
}

func (pr ProposalV0) MarshalZerologObject(e *zerolog.Event) {
	marshalLog(pr, e)
}

func (sb SIGNV0) MarshalZerologObject(e *zerolog.Event) {
	marshalLog(sb, e)
}

func (ab ACCEPTV0) MarshalZerologObject(e *zerolog.Event) {
	marshalLog(ab, e)
}
