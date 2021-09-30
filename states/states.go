package states

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/launch/pm"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util"
)

type States interface {
	util.Daemon
	State() base.State
	NewSeal(seal.Seal) error
	BlockSavedHook() *pm.Hooks
	LastVoteproof() base.Voteproof
	LastINITVoteproof() base.Voteproof
	Handover() Handover
	StartHandover() error
	EndHandover(network.ConnInfo) error
}
