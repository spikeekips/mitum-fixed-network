package base

// VoteproofSet has voteproofs of Ballot, it will be used to deliver multiple
// voteproofs at once.
type VoteproofSet struct {
	Voteproof
	avp Voteproof
}

func NewVoteproofSet(bvp, avp Voteproof) VoteproofSet {
	return VoteproofSet{Voteproof: bvp, avp: avp}
}

func (vp VoteproofSet) ACCEPTVoteproof() Voteproof {
	return vp.avp
}
