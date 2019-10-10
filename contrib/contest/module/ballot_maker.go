package contest_module

var BallotMakers []string

func init() {
	BallotMakers = append(BallotMakers,
		"DefaultBallotMaker",
		"DamangedBallotMaker",
		"ConditionBallotMaker",
	)
}
