package block

import "github.com/rs/zerolog"

func (bm ManifestV0) MarshalZerologObject(e *zerolog.Event) {
	e.
		Stringer("hash", bm.Hash()).
		Int64("height", bm.Height().Int64()).
		Uint64("round", bm.Round().Uint64()).
		Stringer("proposal", bm.Proposal()).
		Stringer("previous", bm.PreviousBlock()).
		Stringer("operations", bm.OperationsHash()).
		Stringer("states", bm.StatesHash())
}
