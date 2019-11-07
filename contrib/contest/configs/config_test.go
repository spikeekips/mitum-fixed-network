package configs

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	contest_module "github.com/spikeekips/mitum/contrib/contest/module"
	"github.com/spikeekips/mitum/isaac"
)

type testMainConfig struct {
	suite.Suite
}

func (t *testMainConfig) TestSuffrage() {
	{
		source := `
global:
  modules:
    suffrage:
      name: FixedProposerSuffrage
      proposer: n0
      number_of_acting: 4
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		t.IsType(&Config{}, nc)

		err = nc.IsValid()
		t.NoError(err)

		fc, ok := nc.Global.Modules.Suffrage.(*contest_module.FixedProposerSuffrageConfig)
		t.True(ok)

		t.Equal("FixedProposerSuffrage", fc.Name())
		t.Equal("n0", fc.Proposer)
		t.Equal(uint(4), fc.NumberOfActing())
	}

	{
		source := `
global:
  modules:
    suffrage:
      name: ConditionSuffrage
      number_of_acting: 4
      conditions:
        - condition: a = 1
          actions:
            - action: do-somthing
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		fc, ok := nc.Global.Modules.Suffrage.(*contest_module.ConditionSuffrageConfig)
		t.True(ok)

		t.Equal("ConditionSuffrage", fc.Name())
		t.Equal(uint(4), fc.NumberOfActing())
		t.Equal(1, len(fc.Conditions))
		t.Equal("a = 1", fc.Conditions[0].ActionChecker().Checker().Query())
	}
}

func (t *testMainConfig) TestInvalidNodeName() {
	{
		source := `
global:
  modules:
    suffrage:
      name: FixedProposerSuffrage
      proposer: n0
      number_of_acting: 4
nodes:
  showme:
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.Contains(err.Error(), "invalid node name")
	}
}

func (t *testMainConfig) TestSuffrageMerge() {
	{
		source := `
global:
  modules:
    suffrage:
      name: FixedProposerSuffrage
      proposer: n0
      number_of_acting: 4
nodes:
  n0:
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		err = nc.Merge(nil)
		t.NoError(err)

		t.Equal("n0", nc.Nodes["n0"].Modules.Suffrage.(*contest_module.FixedProposerSuffrageConfig).Proposer)
	}

	{ // missing number_of_acting in node
		source := `
global:
  modules:
    suffrage:
      name: FixedProposerSuffrage
      proposer: n0
      number_of_acting: 4
nodes:
  n0:
    modules:
      suffrage:
        name: FixedProposerSuffrage
        proposer: n0
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		err = nc.Merge(nil)
		t.NoError(err)

		t.Equal("n0", nc.Nodes["n0"].Modules.Suffrage.(*contest_module.FixedProposerSuffrageConfig).Proposer)
		t.Equal(uint(4), nc.Nodes["n0"].Modules.Suffrage.(*contest_module.FixedProposerSuffrageConfig).NumberOfActing())
	}

	{ // missing number_of_acting in node
		source := `
global:
  modules:
    suffrage:
      name: ConditionSuffrage
      number_of_acting: 4
      conditions:
        - condition: a = 1
          actions:
            - action: do-somthing
nodes:
  n0:
    modules:
      suffrage:
        name: FixedProposerSuffrage
        proposer: n0
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		err = nc.Merge(nil)
		t.NoError(err)

		t.Equal("n0", nc.Nodes["n0"].Modules.Suffrage.(*contest_module.FixedProposerSuffrageConfig).Proposer)
		t.Equal(uint(4), nc.Nodes["n0"].Modules.Suffrage.(*contest_module.FixedProposerSuffrageConfig).NumberOfActing())
	}

	{ // missing number_of_acting in node
		source := `
global:
  modules:
    suffrage:
      name: ConditionSuffrage
      number_of_acting: 4
      conditions:
        - condition: a = 1
          actions:
            - action: do-somthing
nodes:
  n0:
    modules:
      suffrage:
        name: ConditionSuffrage
        conditions:
          - condition: a = 1
            actions:
              - action: do-somthing
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		err = nc.Merge(nil)
		t.NoError(err)

		t.Equal("ConditionSuffrage", nc.Nodes["n0"].Modules.Suffrage.(*contest_module.ConditionSuffrageConfig).Name())
		t.Equal(uint(4), nc.Nodes["n0"].Modules.Suffrage.(*contest_module.ConditionSuffrageConfig).NumberOfActing())
	}

	{ // default number_of_acting
		source := `
global:
  modules:
    suffrage:
      name: FixedProposerSuffrage
      proposer: n0
nodes:
  n0:
`

		nc, err := LoadConfig([]byte(source), 0)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		err = nc.Merge(nil)
		t.NoError(err)

		t.Equal("n0", nc.Nodes["n0"].Modules.Suffrage.(*contest_module.FixedProposerSuffrageConfig).Proposer)
		t.Equal(uint(1), nc.Nodes["n0"].Modules.Suffrage.(*contest_module.FixedProposerSuffrageConfig).NumberOfActing())
	}
}

func (t *testMainConfig) TestPolicy() {
	{
		source := `
global:
  policy:
    threshold: 80
    interval_broadcast_init_ballot_in_join: 3s
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		t.Equal(80.0, *nc.Global.Policy.Threshold)
		t.Equal(time.Second*3, *nc.Global.Policy.IntervalBroadcastINITBallotInJoin)
	}
}

func (t *testMainConfig) TestPolicyMerge() {
	{
		source := `
global:
  policy:
    threshold: 80
    interval_broadcast_init_ballot_in_join: 3s
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		err = nc.Merge(nil)
		t.NoError(err)

		t.Equal(80.0, *nc.Global.Policy.Threshold)
		t.Equal(time.Second*3, *nc.Global.Policy.IntervalBroadcastINITBallotInJoin)
		t.Equal(time.Second*3, *nc.Global.Policy.TimeoutWaitBallot)
		t.Equal(time.Second*3, *nc.Global.Policy.TimeoutWaitINITBallot)
	}
}

func (t *testMainConfig) TestBlocks() {
	{
		source := `
global:
  blocks:
    - height: 10
      round: 1
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		err = nc.Merge(nil)
		t.NoError(err)

		t.Equal(11, len(nc.Global.Blocks))
		t.Equal(isaac.Round(1), *nc.Global.Blocks[10].Round)
		for i, b := range nc.Global.Blocks {
			t.True(isaac.NewBlockHeight(uint64(i)).Equal(*b.Height))
			if isaac.NewBlockHeight(uint64(10)).Equal(*b.Height) {
				//
			} else {
				t.Equal(isaac.Round(0), *b.Round)
			}
		}
	}

	{
		source := `
global:
  blocks:
    - height: 9
      round: 1
    - height: 10
      round: 2
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		err = nc.Merge(nil)
		t.NoError(err)

		t.Equal(11, len(nc.Global.Blocks))
		t.Equal(isaac.Round(1), *nc.Global.Blocks[9].Round)
		t.Equal(isaac.Round(2), *nc.Global.Blocks[10].Round)
		for i, b := range nc.Global.Blocks {
			t.True(isaac.NewBlockHeight(uint64(i)).Equal(*b.Height))
			if isaac.NewBlockHeight(uint64(9)).Equal(*b.Height) {
				//
			} else if isaac.NewBlockHeight(uint64(10)).Equal(*b.Height) {
				//
			} else {
				t.Equal(isaac.Round(0), *b.Round)
			}
		}
	}
}

func (t *testMainConfig) TestBlocksMerge() {
	{
		source := `
global:
  blocks:
    - height: 10
      round: 3

nodes:
  n0:
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		err = nc.Merge(nil)
		t.NoError(err)

		t.Equal(11, len(nc.Nodes["n0"].Blocks))
	}

	{
		source := `
global:
  blocks:
    - height: 10
      round: 3

nodes:
  n0:
    blocks:
      - height: 3
        round: 2
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		err = nc.Merge(nil)
		t.NoError(err)

		t.Equal(11, len(nc.Nodes["n0"].Blocks))
		t.Equal(isaac.Round(2), *nc.Nodes["n0"].Blocks[3].Round)
		t.Equal(isaac.Round(3), *nc.Nodes["n0"].Blocks[10].Round)
		for i, b := range nc.Nodes["n0"].Blocks {
			t.True(isaac.NewBlockHeight(uint64(i)).Equal(*b.Height))
			if isaac.NewBlockHeight(uint64(3)).Equal(*b.Height) {
				//
			} else if isaac.NewBlockHeight(uint64(10)).Equal(*b.Height) {
				//
			} else {
				t.Equal(isaac.Round(0), *b.Round)
			}
		}
	}

	{
		source := `
global:
  blocks:
    - height: 10
      round: 3

nodes:
  n0:
    blocks:
      - height: 3
        round: 2
  n1:
    blocks:
      - height: 20
        round: 4
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		err = nc.Merge(nil)
		t.NoError(err)

		t.Equal(11, len(nc.Nodes["n0"].Blocks))
		t.Equal(isaac.Round(0), *nc.Nodes["n1"].Blocks[3].Round)
		t.Equal(isaac.Round(3), *nc.Nodes["n1"].Blocks[10].Round)
		for i, b := range nc.Nodes["n0"].Blocks {
			t.True(isaac.NewBlockHeight(uint64(i)).Equal(*b.Height))
			if isaac.NewBlockHeight(uint64(3)).Equal(*b.Height) {
				//
			} else if isaac.NewBlockHeight(uint64(10)).Equal(*b.Height) {
				//
			} else {
				t.Equal(isaac.Round(0), *b.Round)
			}
		}

		t.Equal(21, len(nc.Nodes["n1"].Blocks))
		t.Equal(isaac.Round(4), *nc.Nodes["n1"].Blocks[20].Round)
		t.Equal(isaac.Round(0), *nc.Nodes["n1"].Blocks[3].Round)
		t.Equal(isaac.Round(3), *nc.Nodes["n1"].Blocks[10].Round)
		for i, b := range nc.Nodes["n1"].Blocks {
			t.True(isaac.NewBlockHeight(uint64(i)).Equal(*b.Height))

			if isaac.NewBlockHeight(uint64(20)).Equal(*b.Height) {
				//
			} else if isaac.NewBlockHeight(uint64(10)).Equal(*b.Height) {
				//
			} else {
				t.Equal(isaac.Round(0), *b.Round)
			}
		}
	}
}

func (t *testMainConfig) TestModulesMerge() {
	{
		source := `
global:
  modules:
    proposal_maker:
      name: DefaultProposalMaker
nodes:
  n0:
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		err = nc.Merge(nil)
		t.NoError(err)

		fc, ok := nc.Global.Modules.ProposalMaker.(*contest_module.DefaultProposalMakerConfig)
		t.True(ok)
		t.Equal("DefaultProposalMaker", fc.Name())

		n0, ok := nc.Nodes["n0"].Modules.ProposalMaker.(*contest_module.DefaultProposalMakerConfig)
		t.True(ok)
		t.Equal("DefaultProposalMaker", n0.Name())

		t.Equal(fc.Delay(), n0.Delay())
	}

	{ // different dealy
		source := `
global:
  modules:
    proposal_maker:
      name: DefaultProposalMaker
nodes:
  n0:
    modules:
      proposal_maker:
        name: DefaultProposalMaker
        delay: 3s
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		err = nc.Merge(nil)
		t.NoError(err)

		fc, ok := nc.Global.Modules.ProposalMaker.(*contest_module.DefaultProposalMakerConfig)
		t.True(ok)
		t.Equal("DefaultProposalMaker", fc.Name())

		n0, ok := nc.Nodes["n0"].Modules.ProposalMaker.(*contest_module.DefaultProposalMakerConfig)
		t.True(ok)
		t.Equal("DefaultProposalMaker", n0.Name())

		t.NotEqual(fc.Delay(), n0.Delay())
		t.Equal(time.Second*3, n0.Delay())
	}

	{
		source := `
global:
  modules:
    proposal_maker:
      name: DefaultProposalMaker
nodes:
  n0:
    modules:
      proposal_maker:
        name: ConditionProposalMaker
        conditions:
          - condition: a = 1
            actions:
              - action: do-something
                value: hahaha
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		err = nc.Merge(nil)
		t.NoError(err)

		fc, ok := nc.Global.Modules.ProposalMaker.(*contest_module.DefaultProposalMakerConfig)
		t.True(ok)
		t.Equal("DefaultProposalMaker", fc.Name())

		n0, ok := nc.Nodes["n0"].Modules.ProposalMaker.(*contest_module.ConditionProposalMakerConfig)
		t.True(ok)
		t.Equal("ConditionProposalMaker", n0.Name())
	}
}

func (t *testMainConfig) TestConditions() {
	{
		source := `
conditions:
  all:
    - a = 1 AND b = 2
    - c = 1 AND d = 2
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		all, found := nc.Conditions["all"]
		t.True(found)

		t.Equal(2, len(all))
		t.Equal("a = 1 AND b = 2", all[0].Query())
		t.Equal("c = 1 AND d = 2", all[1].Query())
	}
}

func (t *testMainConfig) TestProposalMaker() {
	{
		source := `
global:
  modules:
    proposal_maker:
      name: DefaultProposalMaker
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		fc, ok := nc.Global.Modules.ProposalMaker.(*contest_module.DefaultProposalMakerConfig)
		t.True(ok)

		t.Equal("DefaultProposalMaker", fc.Name())
	}

	{
		source := `
global:
  modules:
    proposal_maker:
      name: ConditionProposalMaker
      conditions:
        - condition: a = 1
          actions:
            - action: do-something
              value: hahaha
        - condition: a = 2
          actions:
            - action: dont-something
              value: hihihi
            - action: make-something
              value: kokoko
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		fc, ok := nc.Global.Modules.ProposalMaker.(*contest_module.ConditionProposalMakerConfig)
		t.True(ok)

		t.Equal("ConditionProposalMaker", fc.Name())
		t.Equal(2, len(fc.Conditions))

		t.Equal("a = 1", fc.Conditions[0].ActionChecker().Checker().Query())
		t.Equal("do-something", fc.Conditions[0].Actions[0].Action)
		t.Equal("hahaha", fc.Conditions[0].Actions[0].Value)
		t.Equal("a = 2", fc.Conditions[1].ActionChecker().Checker().Query())
		t.Equal("dont-something", fc.Conditions[1].Actions[0].Action)
		t.Equal("hihihi", fc.Conditions[1].Actions[0].Value)
		t.Equal("make-something", fc.Conditions[1].Actions[1].Action)
		t.Equal("kokoko", fc.Conditions[1].Actions[1].Value)
	}
}

func (t *testMainConfig) TestProposalValidator() {
	{
		source := `
global:
  modules:
    proposal_validator:
      name: DefaultProposalValidator
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		fc, ok := nc.Global.Modules.ProposalValidator.(*contest_module.DefaultProposalValidatorConfig)
		t.True(ok)

		t.Equal("DefaultProposalValidator", fc.Name())
	}

	{
		source := `
global:
  modules:
    proposal_validator:
      name: ConditionProposalValidator
      conditions:
        - condition: a = 1
          actions:
            - action: do-something
              value: hahaha
        - condition: a = 2
          actions:
            - action: dont-something
              value: hihihi
            - action: make-something
              value: kokoko
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		fc, ok := nc.Global.Modules.ProposalValidator.(*contest_module.ConditionProposalValidatorConfig)
		t.True(ok)

		t.Equal("ConditionProposalValidator", fc.Name())
		t.Equal(2, len(fc.Conditions))

		t.Equal("a = 1", fc.Conditions[0].ActionChecker().Checker().Query())
		t.Equal("do-something", fc.Conditions[0].Actions[0].Action)
		t.Equal("hahaha", fc.Conditions[0].Actions[0].Value)
		t.Equal("a = 2", fc.Conditions[1].ActionChecker().Checker().Query())
		t.Equal("dont-something", fc.Conditions[1].Actions[0].Action)
		t.Equal("hihihi", fc.Conditions[1].Actions[0].Value)
		t.Equal("make-something", fc.Conditions[1].Actions[1].Action)
		t.Equal("kokoko", fc.Conditions[1].Actions[1].Value)
	}
}

func (t *testMainConfig) TestStartCondition() {
	{
		source := `
nodes:
  n0:
    start-condition: node="n1" AND height="13"
`

		nc, err := LoadConfig([]byte(source), 4)
		t.NoError(err)

		err = nc.IsValid()
		t.NoError(err)

		sc := nc.Nodes["n0"].StartCondition
		t.Equal(`node="n1" AND height="13"`, sc.Query())
	}
}

func TestMainConfig(t *testing.T) {
	suite.Run(t, new(testMainConfig))
}
