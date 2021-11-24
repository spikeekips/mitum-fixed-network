//go:build test
// +build test

package base

func (sfs BaseSignedBallotFact) SetFactSign(fs BallotFactSign) BaseSignedBallotFact {
	sfs.factSign = fs

	return sfs
}
