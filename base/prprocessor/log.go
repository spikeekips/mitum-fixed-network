package prprocessor

import "github.com/rs/zerolog"

func LogEventProcessor(pr Processor, key string, e *zerolog.Event) *zerolog.Event {
	if pr == nil {
		return e.Object(key, nil)
	}

	return e.Dict(key, zerolog.Dict().
		Stringer("state", pr.State()).
		Int64("height", pr.Proposal().Height().Int64()).
		Uint64("round", pr.Proposal().Round().Uint64()).
		Stringer("proposal", pr.Proposal().Hash()),
	)
}
