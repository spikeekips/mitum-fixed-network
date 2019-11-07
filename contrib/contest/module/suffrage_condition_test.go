package contest_module

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/contrib/contest/condition"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

type testConditionSuffrage struct {
	suite.Suite
}

func (t *testConditionSuffrage) checkActing(acting isaac.ActingSuffrage) error {
	checked := map[string]bool{}
	for _, n := range acting.Nodes() {
		if _, found := checked[n.Address().String()]; found {
			return xerrors.Errorf("duplicated node found")
		}

		checked[n.Address().String()] = true
	}

	if _, found := checked[acting.Proposer().Address().String()]; !found {
		return xerrors.Errorf("proposer not found")
	}

	return nil
}

func (t *testConditionSuffrage) TestNew() {
	var numberOfNodes uint = 4
	var numberOfActing uint = 3

	var nodes []node.Node
	for i := uint(0); i < numberOfNodes; i++ {
		nodes = append(nodes, node.NewRandomHome())
	}

	lastBlock := NewRandomBlock()
	homeState := isaac.NewHomeState(nodes[0].(node.Home), lastBlock).SetBlock(NewRandomNextBlock(lastBlock))

	query := fmt.Sprintf(`suffrage.height = %s`, homeState.Block().Height())

	cc, _ := condition.NewConditionChecker(query)

	cs := NewConditionSuffrage(
		homeState,
		[]condition.ActionChecker{
			condition.NewActionChecker(cc, condition.NewActionWithoutValue("random")),
		},
		numberOfActing,
		nodes...,
	)

	acting := cs.Acting(homeState.Block().Height(), isaac.Round(0))
	t.Equal(numberOfActing, uint(len(acting.Nodes())))
}

func (t *testConditionSuffrage) TestFixedProposer() {
	var numberOfNodes uint = 4
	var numberOfActing uint = 3

	var nodes []node.Node
	for i := uint(0); i < numberOfNodes; i++ {
		nodes = append(nodes, node.NewRandomHome())
	}

	lastBlock := NewRandomBlock()
	homeState := isaac.NewHomeState(nodes[0].(node.Home), lastBlock).SetBlock(NewRandomNextBlock(lastBlock))

	defaultSuffrage := NewRoundrobinSuffrage(numberOfActing, nodes...)
	defaultActing := defaultSuffrage.Acting(homeState.Block().Height(), isaac.Round(0))
	defaultProposer := defaultActing.Proposer()

	var fixed node.Node
	for _, n := range nodes {
		if n.Equal(defaultProposer) {
			continue
		}
		fixed = n
		break
	}

	{ // not matched
		query := fmt.Sprintf(`suffrage.height!=%s`, homeState.Block().Height())
		cc, _ := condition.NewConditionChecker(query)

		cs := NewConditionSuffrage(
			homeState,
			[]condition.ActionChecker{
				condition.NewActionChecker(
					cc,
					condition.NewAction(
						"fixed-proposer",
						condition.NewActionValue([]interface{}{fixed.Address().String()}, reflect.String),
					),
				),
			},
			numberOfActing,
			nodes...,
		)

		acting := cs.Acting(homeState.Block().Height(), isaac.Round(0))
		t.False(fixed.Equal(acting.Proposer()))
		t.Equal(numberOfActing, uint(len(acting.Nodes())))

		if err := t.checkActing(acting); err != nil {
			t.NoError(err)
		}
	}

	{ // matched
		query := fmt.Sprintf(`suffrage.height = %s`, homeState.Block().Height())
		cc, _ := condition.NewConditionChecker(query)

		cs := NewConditionSuffrage(
			homeState,
			[]condition.ActionChecker{
				condition.NewActionChecker(
					cc,
					condition.NewAction(
						"fixed-proposer",
						condition.NewActionValue([]interface{}{fixed.Address().String()}, reflect.String),
					),
				),
			},
			numberOfActing,
			nodes...,
		)

		acting := cs.Acting(homeState.Block().Height(), isaac.Round(0))
		t.True(fixed.Equal(acting.Proposer()))
		t.Equal(numberOfActing, uint(len(acting.Nodes())))

		if err := t.checkActing(acting); err != nil {
			t.NoError(err)
		}
	}
}

func (t *testConditionSuffrage) TestFixedProposerButUnknownNode() {
	var numberOfNodes uint = 4
	var numberOfActing uint = 3

	var nodes []node.Node
	for i := uint(0); i < numberOfNodes; i++ {
		nodes = append(nodes, node.NewRandomHome())
	}

	lastBlock := NewRandomBlock()
	homeState := isaac.NewHomeState(nodes[0].(node.Home), lastBlock).SetBlock(NewRandomNextBlock(lastBlock))

	fixedName := "show yourself"

	{ // matched
		query := fmt.Sprintf(`suffrage.height = %s`, homeState.Block().Height())
		cc, _ := condition.NewConditionChecker(query)

		cs := NewConditionSuffrage(
			homeState,
			[]condition.ActionChecker{
				condition.NewActionChecker(
					cc,
					condition.NewAction(
						"fixed-proposer",
						condition.NewActionValue([]interface{}{fixedName}, reflect.String),
					),
				),
			},
			numberOfActing,
			nodes...,
		)

		t.Panics(func() {
			cs.Acting(homeState.Block().Height(), isaac.Round(0))
		})
	}
}

func (t *testConditionSuffrage) TestFixedActing() {
	var numberOfNodes uint = 4
	var numberOfActing uint = 3

	var nodes []node.Node
	for i := uint(0); i < numberOfNodes; i++ {
		nodes = append(nodes, node.NewRandomHome())
	}

	lastBlock := NewRandomBlock()
	homeState := isaac.NewHomeState(nodes[0].(node.Home), lastBlock).SetBlock(NewRandomNextBlock(lastBlock))

	defaultSuffrage := NewRoundrobinSuffrage(numberOfActing, nodes...)
	defaultActing := defaultSuffrage.Acting(homeState.Block().Height(), isaac.Round(0))

	var defaultActingNames []string
	for _, n := range defaultActing.Nodes() {
		defaultActingNames = append(defaultActingNames, n.Address().String())
	}

	fixedActingNodes := nodes[:2]
	node.SortNodesByAddress(fixedActingNodes)

	var fixedActingNames []interface{}
	for _, n := range fixedActingNodes {
		fixedActingNames = append(fixedActingNames, n.Address().String())
	}

	{ // not matched
		query := fmt.Sprintf(`suffrage.height != %s`, homeState.Block().Height())
		cc, _ := condition.NewConditionChecker(query)

		cs := NewConditionSuffrage(
			homeState,
			[]condition.ActionChecker{
				condition.NewActionChecker(
					cc,
					condition.NewAction(
						"fixed-acting",
						condition.NewActionValue(fixedActingNames, reflect.String),
					),
				),
			},
			numberOfActing,
			nodes...,
		)

		acting := cs.Acting(homeState.Block().Height(), isaac.Round(0))
		t.Equal(int(numberOfActing), len(acting.Nodes()))

		var actingNodeNames []string
		for _, n := range acting.Nodes() {
			actingNodeNames = append(actingNodeNames, n.Address().String())
		}

		t.Equal(defaultActingNames, actingNodeNames)

		if err := t.checkActing(acting); err != nil {
			t.NoError(err)
		}
	}

	{ // matched
		query := fmt.Sprintf(`suffrage.height = %s`, homeState.Block().Height())
		cc, _ := condition.NewConditionChecker(query)

		cs := NewConditionSuffrage(
			homeState,
			[]condition.ActionChecker{
				condition.NewActionChecker(
					cc,
					condition.NewAction(
						"fixed-acting",
						condition.NewActionValue(fixedActingNames, reflect.String),
					),
				),
			},
			numberOfActing,
			nodes...,
		)

		acting := cs.Acting(homeState.Block().Height(), isaac.Round(0))
		t.Equal(len(fixedActingNames), len(acting.Nodes()))

		var actingNodeNames []interface{}
		for _, n := range acting.Nodes() {
			actingNodeNames = append(actingNodeNames, n.Address().String())
		}

		t.Equal(fixedActingNames, actingNodeNames)

		if err := t.checkActing(acting); err != nil {
			t.NoError(err)
		}
	}
}

func (t *testConditionSuffrage) TestFixedActingButUnknownNodeName() {
	var numberOfNodes uint = 4
	var numberOfActing uint = 3

	var nodes []node.Node
	for i := uint(0); i < numberOfNodes; i++ {
		nodes = append(nodes, node.NewRandomHome())
	}

	lastBlock := NewRandomBlock()
	homeState := isaac.NewHomeState(nodes[0].(node.Home), lastBlock).SetBlock(NewRandomNextBlock(lastBlock))

	fixedActingNodes := nodes[:2]
	node.SortNodesByAddress(fixedActingNodes)

	var fixedActingNames []interface{}
	for _, n := range fixedActingNodes {
		fixedActingNames = append(fixedActingNames, n.Address().String())
	}

	// unknown node name
	fixedActingNames = append(fixedActingNames, "what is this?")

	{ // matched
		query := fmt.Sprintf(`suffrage.height = %s`, homeState.Block().Height())
		cc, _ := condition.NewConditionChecker(query)

		cs := NewConditionSuffrage(
			homeState,
			[]condition.ActionChecker{
				condition.NewActionChecker(
					cc,
					condition.NewAction(
						"fixed-acting",
						condition.NewActionValue(fixedActingNames, reflect.String),
					),
				),
			},
			numberOfActing,
			nodes...,
		)

		t.Panics(
			func() {
				cs.Acting(homeState.Block().Height(), isaac.Round(0))
			},
		)
	}
}

func (t *testConditionSuffrage) TestRandom() {
	var numberOfNodes uint = 10
	var numberOfActing uint = 3

	var nodes []node.Node
	for i := uint(0); i < numberOfNodes; i++ {
		nodes = append(nodes, node.NewRandomHome())
	}

	lastBlock := NewRandomBlock()
	homeState := isaac.NewHomeState(nodes[0].(node.Home), lastBlock).SetBlock(NewRandomNextBlock(lastBlock))

	defaultSuffrage := NewRoundrobinSuffrage(numberOfActing, nodes...)
	defaultActing := defaultSuffrage.Acting(homeState.Block().Height(), isaac.Round(0))

	var defaultActingNames []string
	for _, n := range defaultActing.Nodes() {
		defaultActingNames = append(defaultActingNames, n.Address().String())
	}

	{ // matched
		query := fmt.Sprintf(`suffrage.height = %s`, homeState.Block().Height())
		cc, _ := condition.NewConditionChecker(query)

		cs := NewConditionSuffrage(
			homeState,
			[]condition.ActionChecker{
				condition.NewActionChecker(cc, condition.NewActionWithoutValue("random")),
			},
			numberOfActing,
			nodes...,
		)

		acting := cs.Acting(homeState.Block().Height(), isaac.Round(0))
		t.Equal(len(defaultActing.Nodes()), len(acting.Nodes()))

		var actingNodeNames []interface{}
		for _, n := range acting.Nodes() {
			actingNodeNames = append(actingNodeNames, n.Address().String())
		}

		t.NotEqual(defaultActingNames, actingNodeNames)

		if err := t.checkActing(acting); err != nil {
			t.NoError(err)
		}
	}
}

func TestConditionSuffrage(t *testing.T) {
	suite.Run(t, new(testConditionSuffrage))
}
