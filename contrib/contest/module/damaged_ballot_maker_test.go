package contest_module

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

type testDamagedBallotMaker struct {
	suite.Suite
}

func (t *testDamagedBallotMaker) TestFilter() {
	db := NewDamangedBallotMaker(node.NewRandomHome())

	height := isaac.NewBlockHeight(1)
	round := isaac.Round(0)
	stage := isaac.StageINIT

	db = db.AddPoint(height.String(), round.String(), stage.String())
	kinds := db.IsDamaged(height, round, stage)
	t.NotNil(kinds)

	previousBlock := NewRandomBlockHash()
	newBlock := NewRandomBlockHash()

	{ // INIT
		ballot, err := db.INIT(
			previousBlock,
			isaac.Round(0),
			height,
			newBlock,
			round,
			NewRandomProposalHash(),
		)
		t.NoError(err)
		t.True(ballot.LastBlock().Equal(previousBlock))
		t.False(ballot.Block().Equal(newBlock))
	}

	{ // SIGN
		ballot, err := db.SIGN(
			previousBlock,
			isaac.Round(0),
			height,
			newBlock,
			round,
			NewRandomProposalHash(),
		)
		t.NoError(err)
		t.True(ballot.LastBlock().Equal(previousBlock))
		t.True(ballot.Block().Equal(newBlock))
	}

	{ // ACCEPT
		ballot, err := db.ACCEPT(
			previousBlock,
			isaac.Round(0),
			height,
			newBlock,
			round,
			NewRandomProposalHash(),
		)
		t.NoError(err)
		t.True(ballot.LastBlock().Equal(previousBlock))
		t.True(ballot.Block().Equal(newBlock))
	}
}

func (t *testDamagedBallotMaker) TestKinds() {
	db := NewDamangedBallotMaker(node.NewRandomHome())

	height := isaac.NewBlockHeight(1)
	round := isaac.Round(0)
	stage := isaac.StageINIT

	db = db.AddPoint(
		height.String(),
		round.String(),
		stage.String(),
		"nextBlock", "nextBlock", // duplication will be eliminated
		"currentRound",
	)

	kinds := db.IsDamaged(height, round, stage)
	t.NotNil(kinds)
	t.Equal(kinds, []string{"nextBlock", "currentRound"})

	previousBlock := NewRandomBlockHash()
	newBlock := NewRandomBlockHash()

	{ // INIT
		ballot, err := db.INIT(
			previousBlock,
			isaac.Round(0),
			height,
			newBlock,
			round,
			NewRandomProposalHash(),
		)
		t.NoError(err)
		t.True(ballot.LastBlock().Equal(previousBlock))
		t.False(ballot.Block().Equal(newBlock))
		t.NotEqual(ballot.Round(), round)
	}
}

func (t *testDamagedBallotMaker) TestGlobal() {
	db := NewDamangedBallotMaker(node.NewRandomHome())

	height := isaac.NewBlockHeight(1)
	round := isaac.Round(0)
	stage := isaac.StageINIT

	db = db.AddPoint(
		"",
		"",
		"",
		"nextBlock",
		"currentRound",
	)

	kinds := db.IsDamaged(height, round, stage)
	t.NotNil(kinds)
	t.Equal(kinds, []string{"nextBlock", "currentRound"})

	previousBlock := NewRandomBlockHash()
	newBlock := NewRandomBlockHash()

	{ // INIT
		ballot, err := db.INIT(
			previousBlock,
			isaac.Round(0),
			height,
			newBlock,
			round,
			NewRandomProposalHash(),
		)
		t.NoError(err)
		t.True(ballot.LastBlock().Equal(previousBlock))
		t.False(ballot.Block().Equal(newBlock))
		t.NotEqual(ballot.Round(), round)
	}

	{ // INIT; next height and round
		ballot, err := db.INIT(
			previousBlock,
			isaac.Round(0),
			height.Add(1),
			newBlock,
			round+isaac.Round(1),
			NewRandomProposalHash(),
		)
		t.NoError(err)
		t.True(ballot.LastBlock().Equal(previousBlock))
		t.False(ballot.Block().Equal(newBlock))
		t.NotEqual(ballot.Round(), round+isaac.Round(1))
	}
}

func TestDamagedBallotMaker(t *testing.T) {
	suite.Run(t, new(testDamagedBallotMaker))
}
