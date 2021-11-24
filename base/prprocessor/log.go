package prprocessor

import "github.com/rs/zerolog"

func LogEventProcessor(pr Processor, key string, e *zerolog.Event) *zerolog.Event {
	if pr == nil {
		return e.Object(key, nil)
	}

	return e.Dict(key, zerolog.Dict().
		Stringer("state", pr.State()).
		Int64("height", pr.Fact().Height().Int64()).
		Uint64("round", pr.Fact().Round().Uint64()).
		Stringer("proposal", pr.Fact().Hash()),
	)
}
