package contest_module

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/contrib/contest/condition"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

type testConditionBallotMaker struct {
	suite.Suite
}

func (t *testConditionBallotMaker) TestEmptyBallot() {
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)
	homeState := isaac.NewHomeState(node.NewRandomHome(), lastBlock).SetBlock(nextBlock)

	currentRound := isaac.Round(1)
	currentProposal := NewRandomProposalHash()

	{
		query := fmt.Sprintf(`ballot.next_height=%s`, nextBlock.Height())

		cc, _ := condition.NewConditionChecker(query)
		cb := NewConditionBallotMaker(
			homeState,
			map[string]ConditionHandler{
				"default": NewConditionHandler(cc, "empty-ballot"),
			},
		)

		_, err := cb.INIT(lastBlock.Hash(), lastBlock.Round(), nextBlock.Height(), nextBlock.Hash(), currentRound, currentProposal)
		t.NotNil(err)
	}

	{
		query := fmt.Sprintf(`ballot.next_height=%s`, nextBlock.Height().Add(1))

		cc, _ := condition.NewConditionChecker(query)
		cb := NewConditionBallotMaker(
			homeState,
			map[string]ConditionHandler{
				"default": NewConditionHandler(cc, "empty-ballot"),
			},
		)

		ballot, err := cb.INIT(lastBlock.Hash(), lastBlock.Round(), nextBlock.Height(), nextBlock.Hash(), currentRound, currentProposal)
		t.NoError(err)

		t.True(ballot.Height().Equal(nextBlock.Height()))
		t.Equal(ballot.Round(), currentRound)
		t.Equal(ballot.Stage(), isaac.StageINIT)
		t.True(ballot.Proposal().Equal(currentProposal))
		t.True(ballot.Block().Equal(nextBlock.Hash()))
		t.True(ballot.LastBlock().Equal(lastBlock.Hash()))
		t.Equal(ballot.LastRound(), lastBlock.Round())
	}
}

func (t *testConditionBallotMaker) TestModifyRandom() {
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)
	homeState := isaac.NewHomeState(node.NewRandomHome(), lastBlock).SetBlock(nextBlock)

	currentRound := isaac.Round(1)
	currentProposal := NewRandomProposalHash()

	query := fmt.Sprintf(`ballot.next_height=%s`, nextBlock.Height())
	cc, _ := condition.NewConditionChecker(query)

	cmps := map[string]func(isaac.Ballot) bool{
		"match-last_round": func(ballot isaac.Ballot) bool {
			return ballot.LastRound() == lastBlock.Round()
		},
		"match-current_round": func(ballot isaac.Ballot) bool {
			return ballot.Round() == currentRound
		},
		"match-stage": func(ballot isaac.Ballot) bool {
			return ballot.Stage() == isaac.StageINIT
		},
		"match-last_block": func(ballot isaac.Ballot) bool {
			return ballot.LastBlock().Equal(lastBlock.Hash())
		},
		"match-next_block": func(ballot isaac.Ballot) bool {
			return ballot.Block().Equal(nextBlock.Hash())
		},
		"match-next_height": func(ballot isaac.Ballot) bool {
			return ballot.Height().Equal(nextBlock.Height())
		},
		"match-current_proposal": func(ballot isaac.Ballot) bool {
			return ballot.Proposal().Equal(currentProposal)
		},
	}

	cases := []struct {
		name     string
		action   string
		cmp      string
		expected bool
	}{
		{
			name:     "random-last_block",
			action:   "random-last_block",
			cmp:      "match-last_block",
			expected: false,
		},
		{
			name:     "random-last_round",
			action:   "random-last_round",
			cmp:      "match-last_round",
			expected: false,
		},
		{
			name:     "random-next_height",
			action:   "random-next_height",
			cmp:      "match-next_height",
			expected: false,
		},
		{
			name:     "random-next_block",
			action:   "random-next_block",
			cmp:      "match-next_block",
			expected: false,
		},
		{
			name:     "random-current_round",
			action:   "random-current_round",
			cmp:      "match-current_round",
			expected: false,
		},
		{
			name:     "random-current_proposal",
			action:   "random-current_proposal",
			cmp:      "match-current_proposal",
			expected: false,
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.T().Run(
			c.name,
			func(*testing.T) {
				cb := NewConditionBallotMaker(
					homeState,
					map[string]ConditionHandler{
						"default": NewConditionHandler(cc, c.action),
					},
				)
				ballot, err := cb.INIT(lastBlock.Hash(), lastBlock.Round(), nextBlock.Height(), nextBlock.Hash(), currentRound, currentProposal)
				t.NoError(err)

				for m, f := range cmps {
					result := f(ballot)
					if c.cmp == m {
						t.Equal(c.expected, result, "%d: %v; %v != %v", i, c.name, c.expected, result)
					} else {
						t.True(result, "%d: %v; %v != %v", i, c.name, c.expected, result)
					}
				}
			},
		)
	}
}

func TestConditionBallotMaker(t *testing.T) {
	suite.Run(t, new(testConditionBallotMaker))
}
