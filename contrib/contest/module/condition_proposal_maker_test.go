package contest_module

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/contrib/contest/condition"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

type testConditionProposalMaker struct {
	suite.Suite
}

func (t *testConditionProposalMaker) TestEmptyProposal() {
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)
	homeState := isaac.NewHomeState(node.NewRandomHome(), lastBlock).SetBlock(nextBlock)

	nextHeight := nextBlock.Height()
	nextRound := isaac.Round(1)

	{
		query := fmt.Sprintf(`proposal.height=%s`, nextBlock.Height())

		cc, _ := condition.NewConditionChecker(query)
		cb := NewConditionProposalMaker(
			homeState,
			0, // no delay
			map[string]condition.Action{
				"default": condition.NewActionWithoutValue(cc, "empty-proposal"),
			},
		)

		_, err := cb.Make(nextHeight, nextRound, lastBlock.Hash())
		t.NotNil(err)
	}

	{ // height matched
		query := fmt.Sprintf(`proposal.height=%s`, nextHeight.Add(1))

		cc, _ := condition.NewConditionChecker(query)
		cb := NewConditionProposalMaker(
			homeState,
			0,
			map[string]condition.Action{
				"default": condition.NewActionWithoutValue(cc, "empty-proposal"),
			},
		)

		proposal, err := cb.Make(nextHeight, nextRound, lastBlock.Hash())
		t.NoError(err)

		t.True(proposal.Height().Equal(nextHeight))
		t.Equal(proposal.Round(), nextRound)
		t.True(proposal.LastBlock().Equal(lastBlock.Hash()))
	}

	{ // round matched
		query := fmt.Sprintf(`proposal.round=%s`, nextRound)

		cc, _ := condition.NewConditionChecker(query)
		cb := NewConditionProposalMaker(
			homeState,
			0, // no delay
			map[string]condition.Action{
				"default": condition.NewActionWithoutValue(cc, "empty-proposal"),
			},
		)

		_, err := cb.Make(nextHeight, nextRound, lastBlock.Hash())
		t.NotNil(err)
	}

	{ // last block matched
		query := fmt.Sprintf(`proposal.last_block=%q`, lastBlock.Hash().String())

		cc, _ := condition.NewConditionChecker(query)
		cb := NewConditionProposalMaker(
			homeState,
			0, // no delay
			map[string]condition.Action{
				"default": condition.NewActionWithoutValue(cc, "empty-proposal"),
			},
		)

		_, err := cb.Make(nextHeight, nextRound, lastBlock.Hash())
		t.NotNil(err)
	}
}

func (t *testConditionProposalMaker) TestModifyRandom() {
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)
	homeState := isaac.NewHomeState(node.NewRandomHome(), lastBlock).SetBlock(nextBlock)

	nextHeight := nextBlock.Height()
	nextRound := isaac.Round(1)

	cmps := map[string]func(isaac.Proposal) bool{
		"match-round": func(proposal isaac.Proposal) bool {
			return proposal.Round() == nextRound
		},
		"match-height": func(proposal isaac.Proposal) bool {
			return proposal.Height().Equal(nextHeight)
		},
		"match-last_block": func(proposal isaac.Proposal) bool {
			return proposal.LastBlock().Equal(lastBlock.Hash())
		},
	}

	cases := []struct {
		name     string
		query    string
		action   string
		cmp      string
		expected bool
	}{
		{
			name:     "random-last_block",
			query:    fmt.Sprintf(`proposal.height=%s`, nextHeight),
			action:   "random-last_block",
			cmp:      "match-last_block",
			expected: false,
		},
		{
			name:     "random-round",
			query:    fmt.Sprintf(`proposal.height=%s`, nextHeight),
			action:   "random-round",
			cmp:      "match-round",
			expected: false,
		},
		{
			name:     "random-height",
			query:    fmt.Sprintf(`proposal.height=%s`, nextHeight),
			action:   "random-height",
			cmp:      "match-height",
			expected: false,
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.T().Run(
			c.name,
			func(*testing.T) {
				cc, _ := condition.NewConditionChecker(c.query)

				cp := NewConditionProposalMaker(
					homeState,
					0,
					map[string]condition.Action{
						"default": condition.NewActionWithoutValue(cc, c.action),
					},
				)

				proposal, err := cp.Make(nextHeight, nextRound, lastBlock.Hash())
				t.NoError(err)

				for m, f := range cmps {
					result := f(proposal)
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

func TestConditionProposalMaker(t *testing.T) {
	suite.Run(t, new(testConditionProposalMaker))
}
