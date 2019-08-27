package contest_module

import (
	"fmt"

	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

type DamangedBallotMaker struct {
	isaac.DefaultBallotMaker
	damaged map[ /* Height + Round + Stage*/ string][]string
}

func NewDamangedBallotMaker(home node.Home) DamangedBallotMaker {
	return DamangedBallotMaker{
		DefaultBallotMaker: isaac.NewDefaultBallotMaker(home),
		damaged:            map[string][]string{},
	}
}

func (db DamangedBallotMaker) key(height isaac.Height, round isaac.Round, stage isaac.Stage) string {
	return fmt.Sprintf("%s-%s-%s", height.String(), round.String(), stage.String())
}

func (db DamangedBallotMaker) AddPoint(
	height isaac.Height,
	round isaac.Round,
	stage isaac.Stage,
	kinds ...string,
) DamangedBallotMaker {
	var nk []string
	seen := map[string]struct{}{}
	for _, k := range kinds {
		if _, found := seen[k]; found {
			continue
		}
		nk = append(nk, k)
		seen[k] = struct{}{}
	}

	k := db.key(height, round, stage)
	db.damaged[k] = nk

	return db
}

func (db DamangedBallotMaker) IsDamaged(height isaac.Height, round isaac.Round, stage isaac.Stage) []string {
	key := db.key(height, round, stage)
	p, found := db.damaged[key]
	if !found {
		return nil
	}

	return p
}

func (db DamangedBallotMaker) INIT(
	previousBlock hash.Hash,
	newBlock hash.Hash,
	newRound isaac.Round,
	newProposal hash.Hash,
	nextHeight isaac.Height,
	nextRound isaac.Round,
) (isaac.Ballot, error) {
	if p := db.IsDamaged(nextHeight, nextRound, isaac.StageINIT); p != nil {
		for _, k := range p {
			switch k {
			case "previousBlock":
				previousBlock = NewRandomBlockHash()
			case "newBlock":
				newBlock = NewRandomBlockHash()
			case "newRound":
				newRound = NewRandomRound()
			case "newProposal":
				newProposal = NewRandomProposalHash()
			case "nextHeight":
				nextHeight = NewRandomHeight()
			case "nextRound":
				nextRound = NewRandomRound()
			}
		}
	}

	return db.DefaultBallotMaker.INIT(
		previousBlock, newBlock, newRound, newProposal, nextHeight, nextRound,
	)
}

func (db DamangedBallotMaker) SIGN(
	lastBlock hash.Hash,
	lastRound isaac.Round,
	nextHeight isaac.Height,
	nextBlock hash.Hash,
	currentRound isaac.Round,
	currentProposal hash.Hash,
) (isaac.Ballot, error) {
	if p := db.IsDamaged(nextHeight, currentRound, isaac.StageSIGN); p != nil {
		for _, k := range p {
			switch k {
			case "lastBlock":
				lastBlock = NewRandomBlockHash()
			case "lastRound":
				lastRound = NewRandomRound()
			case "nextHeight":
				nextHeight = NewRandomHeight()
			case "nextBlock":
				nextBlock = NewRandomBlockHash()
			case "currentRound":
				currentRound = NewRandomRound()
			case "currentProposal":
				currentProposal = NewRandomProposalHash()
			}
		}
	}

	return db.DefaultBallotMaker.SIGN(
		lastBlock, lastRound, nextHeight, nextBlock, currentRound, currentProposal,
	)
}

func (db DamangedBallotMaker) ACCEPT(
	lastBlock hash.Hash,
	lastRound isaac.Round,
	nextHeight isaac.Height,
	nextBlock hash.Hash,
	currentRound isaac.Round,
	currentProposal hash.Hash,
) (isaac.Ballot, error) {
	if p := db.IsDamaged(nextHeight, currentRound, isaac.StageACCEPT); p != nil {
		for _, k := range p {
			switch k {
			case "lastBlock":
				lastBlock = NewRandomBlockHash()
			case "lastRound":
				lastRound = NewRandomRound()
			case "nextHeight":
				nextHeight = NewRandomHeight()
			case "nextBlock":
				nextBlock = NewRandomBlockHash()
			case "currentRound":
				currentRound = NewRandomRound()
			case "currentProposal":
				currentProposal = NewRandomProposalHash()
			}
		}
	}

	return db.DefaultBallotMaker.ACCEPT(
		lastBlock, lastRound, nextHeight, nextBlock, currentRound, currentProposal,
	)
}
