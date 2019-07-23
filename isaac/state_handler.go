package isaac

import (
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/node"
)

type StateHandler interface {
	common.Daemon
	State() node.State
	SetChanState(chan node.State) StateHandler
	ReceiveVoteResult(VoteResult) error
	ReceiveProposal(Proposal) error
}
