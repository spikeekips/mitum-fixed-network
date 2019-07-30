package isaac

import "time"

type Policy struct {
	Threshold                         uint          // base percent for `Threshold`
	IntervalBroadcastINITBallotInJoin time.Duration // interval to broadcast INIT ballot in join
	TimeoutWaitVoteResultInJoin       time.Duration // wait VoteResult in join state
	TimeoutWaitBallot                 time.Duration // wait the new Proposal
}
