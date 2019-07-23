package isaac

import "time"

type Policy struct {
	IntervalBroadcastINITBallotInJoin time.Duration
	TimeoutWaitVoteResultInJoin       time.Duration
}
