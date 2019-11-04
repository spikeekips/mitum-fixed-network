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

	{
		query := fmt.Sprintf(`proposal.height=%s`, nextBlock.Height())

		cc, _ := condition.NewConditionChecker(query)
		cb := NewConditionProposalValidator(
			homeState,
			[]condition.ActionChecker{
				condition.NewActionChecker0(cc, condition.NewActionWithoutValue("empty-proposal")),
			},
		)

		validated := cb.Validated(nextBlock.Proposal())
		t.False(validated)

		block, err := cb.NewBlock(nextBlock.Height(), nextBlock.Round(), nextBlock.Proposal())
		t.NoError(err)

		t.True(nextBlock.Height().Equal(block.Height()))
		t.Equal(nextBlock.Round(), block.Round())

		validated = cb.Validated(nextBlock.Proposal())
		t.True(validated)
	}

	{ // `fail` action: failed to make new block
		query := fmt.Sprintf(`block.height=%s`, nextBlock.Height())

		cc, _ := condition.NewConditionChecker(query)
		cb := NewConditionProposalValidator(
			homeState,
			[]condition.ActionChecker{
				condition.NewActionChecker0(cc, condition.NewActionWithoutValue("fail")),
			},
		)

		validated := cb.Validated(nextBlock.Proposal())
		t.False(validated)

		_, err := cb.NewBlock(nextBlock.Height(), nextBlock.Round(), nextBlock.Proposal())
		t.Contains(err.Error(), "failed to make new block")
	}

	{ // `block-hash` action: set block hash
		newBlockHash := NewRandomBlockHash()
		query := fmt.Sprintf(`block.height=%s`, nextBlock.Height())

		cc, _ := condition.NewConditionChecker(query)
		cb := NewConditionProposalValidator(
			homeState,
			[]condition.ActionChecker{
				condition.NewActionChecker0(cc, condition.NewAction(
					"block-hash",
					condition.NewActionValue(
						[]interface{}{newBlockHash.String()},
						reflect.String,
					),
				)),
			},
		)

		validated := cb.Validated(nextBlock.Proposal())
		t.False(validated)

		block, err := cb.NewBlock(nextBlock.Height(), nextBlock.Round(), nextBlock.Proposal())
		t.NoError(err)
		t.True(newBlockHash.Equal(block.Hash()))

		t.True(nextBlock.Height().Equal(block.Height()))
		t.Equal(nextBlock.Round(), block.Round())
	}

	{ // `random-block-hash` action: set random block hash
		var correctBlockHash hash.Hash
		{
			cb := NewConditionProposalValidator(homeState, nil)
			block, err := cb.NewBlock(nextBlock.Height(), nextBlock.Round(), nextBlock.Proposal())
			t.NoError(err)
			correctBlockHash = block.Hash()
		}

		query := fmt.Sprintf(`block.height=%s`, nextBlock.Height())

		cc, _ := condition.NewConditionChecker(query)
		cb := NewConditionProposalValidator(
			homeState,
			[]condition.ActionChecker{
				condition.NewActionChecker0(cc, condition.NewActionWithoutValue("random-block-hash")),
			},
		)

		validated := cb.Validated(nextBlock.Proposal())
		t.False(validated)

		block, err := cb.NewBlock(nextBlock.Height(), nextBlock.Round(), nextBlock.Proposal())
		t.NoError(err)
		t.False(correctBlockHash.Equal(block.Hash()))
	}
}

func TestConditionProposalValidator(t *testing.T) {
	suite.Run(t, new(testConditionProposalValidator))
}
