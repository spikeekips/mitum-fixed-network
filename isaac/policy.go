package isaac

import "time"

type Policy struct {
	IntervalBroadcastINITBallotInJoin time.Duration // interval to broadcast INIT ballot in join
	TimeoutWaitVoteResultInJoin       time.Duration // wait VoteResult in join state
	timeoutWaitBallot                 time.Duration // wait the new Proposal
}
