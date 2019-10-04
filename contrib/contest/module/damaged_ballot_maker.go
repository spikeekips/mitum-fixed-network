package contest_module

import (
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
	"golang.org/x/xerrors"
)

type DamangedBallotMaker struct {
	isaac.DefaultBallotMaker
	*common.Logger
	damaged map[ /* Height + Round + Stage*/ string][]string
}

func NewDamangedBallotMaker(home node.Home) DamangedBallotMaker {
	return DamangedBallotMaker{
		DefaultBallotMaker: isaac.NewDefaultBallotMaker(home),
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "damanged-ballot_maker")
		}),
		damaged: map[string][]string{},
	}
}

func (db DamangedBallotMaker) key(height, round, stage string) string {
	return fmt.Sprintf("%s-%s-%s", height, round, stage)
}

func (db DamangedBallotMaker) AddPoint(height, round, stage string, kinds ...string) DamangedBallotMaker {
	nk := []string{}

	if len(height) < 1 {
		height = "*"
	}
	if len(round) < 1 {
		round = "*"
	}
	if len(stage) < 1 {
		stage = "*"
	}

	if len(kinds) > 0 {
		seen := map[string]struct{}{}
		for _, k := range kinds {
			if _, found := seen[k]; found {
				continue
			}
			nk = append(nk, k)
			seen[k] = struct{}{}
		}
	}

	k := db.key(height, round, stage)
	db.damaged[k] = nk

	return db
}

func (db DamangedBallotMaker) IsDamaged(height isaac.Height, round isaac.Round, stage isaac.Stage) []string {
	keys := []string{
		db.key(height.String(), round.String(), stage.String()), // exact match
		db.key(height.String(), "*", "*"),                       // height match
		db.key("*", "*", stage.String()),                        // stage match
		db.key("*", "*", "*"),                                   // global match
	}

	for _, k := range keys {
		p := db.isDamaged(k)
		if p != nil {
			return p
		}
	}

	return nil
}

func (db DamangedBallotMaker) isDamaged(key string) []string {
	p, found := db.damaged[key]
	if !found {
		return nil
	}

	return p
}

func (db DamangedBallotMaker) modifyBallot(
	lastBlock hash.Hash,
	lastRound isaac.Round,
	nextHeight isaac.Height,
	nextBlock hash.Hash,
	currentRound isaac.Round,
	currentProposal hash.Hash,
	stage isaac.Stage,
) (isaac.Ballot, error) {
	if p := db.IsDamaged(nextHeight, currentRound, stage); p != nil {
		db.Log().Debug().
			Uint64("height", nextHeight.Uint64()).
			Uint64("round", currentRound.Uint64()).
			Str("stage", stage.String()).
			Interface("kinds", p).
			Msg("damaged point")

		if len(p) < 1 {
			nextBlock = NewRandomBlockHash()
		} else {
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
	}

	var cbFunc func(hash.Hash, isaac.Round, isaac.Height, hash.Hash, isaac.Round, hash.Hash) (isaac.Ballot, error)

	switch stage {
	case isaac.StageINIT:
		cbFunc = db.DefaultBallotMaker.INIT
	case isaac.StageSIGN:
		cbFunc = db.DefaultBallotMaker.SIGN
	case isaac.StageACCEPT:
		cbFunc = db.DefaultBallotMaker.ACCEPT
	default:
		err := xerrors.Errorf("unknown stage found")
		db.Log().Error().
			Err(err).
			Str("stage", stage.String()).
			Send()
		return isaac.Ballot{}, err
	}

	return cbFunc(
		lastBlock, lastRound, nextHeight, nextBlock, currentRound, currentProposal,
	)
}

func (db DamangedBallotMaker) INIT(
	lastBlock hash.Hash,
	lastRound isaac.Round,
	nextHeight isaac.Height,
	nextBlock hash.Hash,
	currentRound isaac.Round,
	currentProposal hash.Hash,
) (isaac.Ballot, error) {
	return db.modifyBallot(
		lastBlock, lastRound, nextHeight, nextBlock, currentRound, currentProposal,
		isaac.StageINIT,
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
	return db.modifyBallot(
		lastBlock, lastRound, nextHeight, nextBlock, currentRound, currentProposal,
		isaac.StageSIGN,
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
	return db.modifyBallot(
		lastBlock, lastRound, nextHeight, nextBlock, currentRound, currentProposal,
		isaac.StageACCEPT,
	)
}

func (db DamangedBallotMaker) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":    "DamangedBallotMaker",
		"damaged": db.damaged,
	})
}
