package mitum

import (
	"github.com/spikeekips/mitum/logging"
)

/*
ConsensusStateConsensusHandler tries to join network safely.

What does consensus state means?

- Block states are synced with the network.
- Node can participate every vote stages.

Consensus state is started by next INIT VoteResult and waits next Proposal.
*/
type ConsensusStateConsensusHandler struct {
	*logging.Logger
}
