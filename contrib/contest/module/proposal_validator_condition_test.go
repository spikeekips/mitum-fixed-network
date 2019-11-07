package contest_module

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/spikeekips/mitum/contrib/contest/condition"
	"github.com/spikeekips/mitum/hash"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

type testConditionProposalValidator struct {
	suite.Suite
}

func (t *testConditionProposalValidator) TestEmptyProposal() {
	lastBlock := NewRandomBlock()
	nextBlock := NewRandomNextBlock(lastBlock)
	homeState := isaac.NewHomeState(node.NewRandomHome(), lastBlock).SetBlock(nextBlock)

	proposal, _ := isaac.NewProposal(
		nextBlock.Height().Add(1),
		isaac.Round(0),
		nextBlock.Hash(),
		homeState.Home().Address(),
		nil,
	)
	_ = proposal.Sign(homeState.Home().PrivateKey(), nil)
	ss := NewMemorySealStorage()
	_ = ss.Save(proposal)

	{
		query := fmt.Sprintf(`proposal.height=%s`, proposal.Height())

		cc, _ := condition.NewConditionChecker(query)
		cb := NewConditionProposalValidator(
			homeState,
			ss,
			[]condition.ActionChecker{
				condition.NewActionChecker(cc, condition.NewActionWithoutValue("empty-proposal")),
			},
		)

		validated := cb.Validated(proposal.Hash())
		t.False(validated)

		block, err := cb.NewBlock(proposal.Hash())
		t.NoError(err)

		t.True(proposal.Height().Equal(block.Height()))
		t.Equal(proposal.Round(), block.Round())

		validated = cb.Validated(proposal.Hash())
		t.True(validated)
	}

	{ // `fail` action: failed to make new block
		query := fmt.Sprintf(`block.height=%s`, proposal.Height())

		cc, _ := condition.NewConditionChecker(query)
		cb := NewConditionProposalValidator(
			homeState,
			ss,
			[]condition.ActionChecker{
				condition.NewActionChecker(cc, condition.NewAction(
					"error",
					condition.NewActionValue(
						[]interface{}{"show me"},
						reflect.String,
					),
				)),
			},
		)

		validated := cb.Validated(proposal.Hash())
		t.False(validated)

		_, err := cb.NewBlock(proposal.Hash())
		t.Contains(err.Error(), "show me")
	}

	{ // `block-hash` action: set block hash
		newBlockHash := NewRandomBlockHash()
		query := fmt.Sprintf(`block.height=%s`, proposal.Height())

		cc, _ := condition.NewConditionChecker(query)
		cb := NewConditionProposalValidator(
			homeState,
			ss,
			[]condition.ActionChecker{
				condition.NewActionChecker(cc, condition.NewAction(
					"block-hash",
					condition.NewActionValue(
						[]interface{}{newBlockHash.String()},
						reflect.String,
					),
				)),
			},
		)

		validated := cb.Validated(proposal.Hash())
		t.False(validated)

		block, err := cb.NewBlock(proposal.Hash())
		t.NoError(err)
		t.True(newBlockHash.Equal(block.Hash()))

		t.True(proposal.Height().Equal(block.Height()))
		t.Equal(proposal.Round(), block.Round())
	}

	{ // `random-block-hash` action: set random block hash
		var correctBlockHash hash.Hash
		{
			cb := NewConditionProposalValidator(homeState, ss, nil)
			block, err := cb.NewBlock(proposal.Hash())
			t.NoError(err)
			correctBlockHash = block.Hash()
		}

		query := fmt.Sprintf(`block.height=%s`, proposal.Height())

		cc, _ := condition.NewConditionChecker(query)
		cb := NewConditionProposalValidator(
			homeState,
			ss,
			[]condition.ActionChecker{
				condition.NewActionChecker(cc, condition.NewActionWithoutValue("random-block-hash")),
			},
		)

		validated := cb.Validated(proposal.Hash())
		t.False(validated)

		block, err := cb.NewBlock(proposal.Hash())
		t.NoError(err)
		t.False(correctBlockHash.Equal(block.Hash()))

		// same height, round and proposal, new block should be same
		sameBlock, err := cb.NewBlock(proposal.Hash())
		t.NoError(err)

		t.True(block.Hash().Equal(sameBlock.Hash()))
		t.True(block.Proposal().Equal(sameBlock.Proposal()))
	}
}

func TestConditionProposalValidator(t *testing.T) {
	suite.Run(t, new(testConditionProposalValidator))
}
