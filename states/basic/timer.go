package basicstates

import "github.com/spikeekips/mitum/util/localtime"

const (
	TimerIDBroadcastINITBallot         localtime.TimerID = "broadcast-init-ballot"
	TimerIDBroadcastJoingingINITBallot localtime.TimerID = "broadcast-joining-init-ballot"
	TimerIDBroadcastProposal           localtime.TimerID = "broadcast-proposal"
	TimerIDBroadcastACCEPTBallot       localtime.TimerID = "broadcast-accept-ballot"
	TimerIDSyncingWaitVoteproof        localtime.TimerID = "syncing-wait-new-voteproof"
	TimerIDFindProposal                localtime.TimerID = "find-proposal"
)
