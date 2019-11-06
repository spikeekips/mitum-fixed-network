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
	contest_config "github.com/spikeekips/mitum/contrib/contest/config"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

func init() {
	Suffrages = append(Suffrages, "ConditionSuffrage")
	SuffrageConfigs["ConditionSuffrage"] = ConditionSuffrageConfig{}
}

type ConditionSuffrageConfig struct {
	N          string                            `yaml:"name"`
	NA         uint                              `yaml:"number_of_acting,omitempty"`
	Conditions []*contest_config.ActionCondition `yaml:"conditions"`
}

func (cc ConditionSuffrageConfig) Name() string {
	return cc.N
}

func (cc ConditionSuffrageConfig) NumberOfActing() uint {
	return cc.NA
}

func (cc *ConditionSuffrageConfig) IsValid() error {
	// NOTE empty condition be allowed.
	// if len(cc.Conditions) < 1 {
	// 	return xerrors.Errorf("empty `conditions`")
	// }

	var cs []*contest_config.ActionCondition
	for _, c := range cc.Conditions {
		if err := c.IsValid(); err != nil {
			return err
		}
		cs = append(cs, c)
	}

	cc.Conditions = cs

	return nil
}

func (cc *ConditionSuffrageConfig) Merge(i interface{}) error {
	n, ok := interface{}(i).(SuffrageConfig)
	if !ok {
		return xerrors.Errorf("invalid merge source found: %%", i)
	}

	if cc.NA < 1 {
		cc.NA = n.NumberOfActing()
	}

	return nil
}

func (cc *ConditionSuffrageConfig) New(homeState *isaac.HomeState, nodes []node.Node, l zerolog.Logger) isaac.Suffrage {
	var checkers []condition.ActionChecker
	for _, c := range cc.Conditions {
		checkers = append(checkers, c.ActionChecker())
	}

	sf := NewConditionSuffrage(
		homeState,
		checkers,
		cc.NumberOfActing(),
		nodes...,
	)
	sf.SetLogger(l)

	return sf
}

type ConditionSuffrage struct {
	*common.Logger
	*RoundrobinSuffrage
	homeState *isaac.HomeState
	checkers  []condition.ActionChecker
	nameMap   map[string]node.Node
}

func NewConditionSuffrage(
	homeState *isaac.HomeState,
	checkers []condition.ActionChecker,
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
		homeState: homeState,
		checkers:  checkers,
		nameMap:   nameMap,
	}
}

func (cs ConditionSuffrage) NumberOfActing() uint {
	return cs.RoundrobinSuffrage.NumberOfActing()
}

func (cs ConditionSuffrage) Acting(height isaac.Height, round isaac.Round) isaac.ActingSuffrage {
	if cs.checkers == nil {
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

	for _, c := range cs.checkers {
		if c.Checker().Check(li) {
			for _, action := range c.Actions() {
				cs.Log().Debug().
					Str("query", c.Checker().Query()).
					Str("action", action.Action()).
					RawJSON("data", li.Bytes()).
					Msg("condition matched")

				switch t := action.Action(); t {
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
					if names, err := nodeNameFromActionValue(action.Value()); err != nil {
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

					if int(cs.RoundrobinSuffrage.numberOfActing) != len(cs.Nodes()) {
						rand.Seed(time.Now().UnixNano())
						rand.Shuffle(len(rd), func(i, j int) { rd[i], rd[j] = rd[j], rd[i] })
					}

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
					names, err := nodeNameFromActionValue(action.Value())
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
	}

	return cs.RoundrobinSuffrage.Acting(height, round)
}

func (cs ConditionSuffrage) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":                "ConditionSuffrage",
		"checkers":            cs.checkers,
		"roundrobin_suffrage": cs.RoundrobinSuffrage,
	})
}

func nodeNameFromActionValue(ca condition.ActionValue) ([]string, error) {
	if ca.Hint() != reflect.String {
		return nil, xerrors.Errorf("invalid action value type for node name action: %v", ca.Hint())
	}

	var values []string
	for _, i := range ca.Value() {
		values = append(values, i.(string))
	}

	return values, nil
}
