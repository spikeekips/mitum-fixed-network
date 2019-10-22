package contest_module

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"time"

	"golang.org/x/xerrors"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/contrib/contest/condition"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

type ConditionSuffrage struct {
	*common.Logger
	*RoundrobinSuffrage
	homeState  *isaac.HomeState
	conditions map[string]condition.Action
	nameMap    map[string]node.Node
}

func NewConditionSuffrage(
	homeState *isaac.HomeState,
	conditions map[string]condition.Action,
	numberOfActing uint,
	nodes ...node.Node,
) *ConditionSuffrage {
	if int(numberOfActing) > len(nodes) {
		panic(xerrors.Errorf(
			"numberOfActing should be lesser than number of nodes: numberOfActing=%v nodes=%v",
			numberOfActing,
			len(nodes),
		))
	}

	nameMap := map[string]node.Node{}
	for _, n := range nodes {
		name := n.Address().String()
		nameMap[name] = n

		if len(n.Alias()) > 0 {
			nameMap[n.Alias()] = n
		}
	}

	return &ConditionSuffrage{
		RoundrobinSuffrage: NewRoundrobinSuffrage(numberOfActing, nodes...),
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "condition-suffrage")
		}),
		homeState:  homeState,
		conditions: conditions,
		nameMap:    nameMap,
	}
}

func (cs ConditionSuffrage) Acting(height isaac.Height, round isaac.Round) isaac.ActingSuffrage {
	if cs.conditions == nil {
		return cs.RoundrobinSuffrage.Acting(height, round)
	}

	li, err := condition.NewLogItemFromMap(
		map[string]interface{}{
			"node":  cs.homeState.Home().Alias(),
			"state": cs.homeState.State().String(),
			"block": map[string]interface{}{
				"height":   cs.homeState.Block().Height().Uint64(),
				"round":    cs.homeState.Block().Round().Uint64(),
				"proposal": cs.homeState.Block().Proposal().String(),
			},
			"previousBlock": map[string]interface{}{
				"height":   cs.homeState.PreviousBlock().Height().Uint64(),
				"round":    cs.homeState.PreviousBlock().Round().Uint64(),
				"proposal": cs.homeState.PreviousBlock().Proposal().String(),
			},
			"suffrage": map[string]interface{}{
				"height": height.Uint64(),
				"round":  round.Uint64(),
			},
		})
	if err != nil {
		cs.Log().Error().
			Err(err).
			Msg("failed to create LogItem")
		return cs.RoundrobinSuffrage.Acting(height, round)
	}

	for name, c := range cs.conditions {
		if c.Checker().Check(li) {
			cs.Log().Debug().
				Str("checker", name).
				Str("query", c.Checker().Query()).
				Str("action", c.Action()).
				RawJSON("data", li.Bytes()).
				Msg("condition matched")

			switch t := c.Action(); t {
			case "empty-suffrage":
				return isaac.NewActingSuffrage(height, round, nil, nil)
			case "random":
				if int(cs.RoundrobinSuffrage.numberOfActing) == len(cs.Nodes()) {
					return cs.RoundrobinSuffrage.Acting(height, round)
				}

				rd := make([]node.Node, len(cs.Nodes()))
				copy(rd, cs.Nodes())

				rand.Seed(time.Now().UnixNano())
				rand.Shuffle(len(rd), func(i, j int) { rd[i], rd[j] = rd[j], rd[i] })

				return isaac.NewActingSuffrage(
					height,
					round,
					rd[0],
					rd[:cs.RoundrobinSuffrage.numberOfActing],
				)
			case "fixed-proposer":
				var proposer node.Node
				if names, err := nodeNameFromActionValue(c.Value()); err != nil {
					panic(err)
				} else if len(names) != 1 {
					panic(xerrors.Errorf("one value should be set for fixed-proposer"))
				} else if n, found := cs.nameMap[names[0]]; !found {
					panic(xerrors.Errorf("proposer not found for fixed-proposer: %q", names[0]))
				} else {
					proposer = n
				}

				rd := make([]node.Node, len(cs.Nodes()))
				copy(rd, cs.Nodes())

				rand.Seed(time.Now().UnixNano())
				rand.Shuffle(len(rd), func(i, j int) { rd[i], rd[j] = rd[j], rd[i] })

				if proposer == nil {
					panic(xerrors.Errorf(
						"selected proposer not found in nodes for fixed-proposer; %s",
					))
				}

				var nn []node.Node
				nn = append(nn, proposer)
				for _, n := range rd {
					if len(nn) == int(cs.RoundrobinSuffrage.numberOfActing) {
						break
					}

					if n.Equal(proposer) {
						continue
					}

					nn = append(nn, n)
				}

				return isaac.NewActingSuffrage(height, round, proposer, nn)
			case "fixed-acting":
				names, err := nodeNameFromActionValue(c.Value())
				if err != nil {
					panic(err)
				} else if len(names) < 1 {
					panic(xerrors.Errorf("at least one node should be set for fixed-acting"))
				}

				var nodes []node.Node
				for _, name := range names {
					if n, found := cs.nameMap[name]; !found {
						panic(xerrors.Errorf("proposer not found for fixed-acting: %q", name))
					} else {
						nodes = append(nodes, n)
					}
				}

				return isaac.NewActingSuffrage(height, round, nodes[0], nodes)
			default:
				panic(xerrors.Errorf("unknown condition action found: %v", t))
			}
		}
	}

	return cs.RoundrobinSuffrage.Acting(height, round)
}

func (cs ConditionSuffrage) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":                "ConditionSuffrage",
		"conditions":          cs.conditions,
		"roundrobin_suffrage": cs.RoundrobinSuffrage,
	})
}

func nodeNameFromActionValue(value condition.ActionValue) ([]string, error) {
	if value.Hint() != reflect.String {
		return nil, xerrors.Errorf("invalid action value type for node name action: %v", value.Hint())
	}

	var names []string
	for _, v := range value.Value() {
		if p, ok := v.(string); !ok {
			return nil, xerrors.Errorf("invalid action value for node name action: %q", v)
		} else if len(p) < 1 {
			return nil, xerrors.Errorf("empty value for node name")
		} else {
			names = append(names, p)
		}
	}

	return names, nil
}
