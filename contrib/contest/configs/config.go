package configs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/contrib/contest/condition"
	contest_config "github.com/spikeekips/mitum/contrib/contest/config"
	contest_module "github.com/spikeekips/mitum/contrib/contest/module"
	"github.com/spikeekips/mitum/isaac"
)

var blockGenerator sync.Map

type Config struct {
	Global        *NodeConfig
	Nodes         map[string]*NodeConfig
	Conditions    *ConditionsConfig
	NumberOfNodes uint `yaml:"-"`
}

func LoadConfigFromFile(f string, numberOfNodes uint) (*Config, error) {
	log.Debug().
		Uint("number_of_nodes", numberOfNodes).
		Str("file", f).
		Msg("trying to load config")

	b, err := ioutil.ReadFile(filepath.Clean(f))
	if err != nil {
		return nil, xerrors.Errorf("failed to load config(%s): %w", f, err)
	}

	return LoadConfig(b, numberOfNodes)
}

func LoadConfig(b []byte, numberOfNodes uint) (*Config, error) {
	var config Config
	if err := yaml.Unmarshal(b, &config); err != nil {
		return nil, err
	}

	config.NumberOfNodes = numberOfNodes

	return &config, nil
}

func (nc *Config) IsValid() error {
	if nc.Global != nil {
		if err := nc.Global.IsValid(); err != nil {
			return err
		}
	}

	var last uint
	for name, n := range nc.Nodes {
		var i uint
		if _, err := fmt.Sscanf(name, "n%d", &i); err != nil {
			return xerrors.Errorf("invalid node name found, node name should be 'n<number>'; %w", err)
		} else if i > last {
			last = i
		}

		if n == nil {
			continue
		}

		if err := n.IsValid(); err != nil {
			return err
		}
	}

	if nc.NumberOfNodes < 1 {
		nc.NumberOfNodes = last + 1
	}

	if nc.NumberOfNodes < 1 {
		return xerrors.Errorf("not enough nodes")
	}

	if nc.Conditions != nil {
		if err := nc.Conditions.IsValid(); err != nil {
			return err
		}
	}

	return nil
}

func (nc *Config) Merge(interface{}) error {
	if nc.Global == nil {
		nc.Global = defaultNodeConfig(nc.NumberOfNodes)
	} else if err := nc.Global.Merge(defaultNodeConfig(nc.NumberOfNodes)); err != nil {
		return err
	}

	// nodes
	if nc.Nodes == nil {
		nc.Nodes = map[string]*NodeConfig{}
	}

	for i := uint(0); i < nc.NumberOfNodes; i++ {
		name := fmt.Sprintf("n%d", i)
		c, found := nc.Nodes[name]
		if !found {
			c = nc.Global
		} else if c == nil {
			c = nc.Global
		} else if err := merge(c, nc.Global); err != nil {
			return err
		}
		nc.Nodes[name] = c
	}

	return nil
}

func (nc *Config) MarshalZerologObject(e *zerolog.Event) {
	e.Interface("global", nc.Global)
	e.Interface("nodes", nc.Nodes)
	e.Interface("conditions", nc.Conditions)
	e.Uint("number_of_nodes", nc.NumberOfNodes)
}

type NodeConfig struct {
	Policy  *PolicyConfig  `yaml:",omitempty"`
	Blocks  []*BlockConfig `yaml:"blocks,omitempty"`
	Modules *ModulesConfig `yaml:"modules,omitempty"`
}

func defaultNodeConfig(numberOfNodes uint) *NodeConfig {
	return &NodeConfig{
		Policy:  defaultPolicyConfig(),
		Blocks:  defaultNBlocksConfig(10),
		Modules: defaultModulesConfig(numberOfNodes),
	}
}

func (nc *NodeConfig) IsValid() error {
	if nc.Policy != nil {
		if err := nc.Policy.IsValid(); err != nil {
			return err
		}
	}

	for _, b := range nc.Blocks {
		if err := b.IsValid(); err != nil {
			return err
		}
	}

	if nc.Modules != nil {
		if err := nc.Modules.IsValid(); err != nil {
			return err
		}
	}

	return nil
}

func (nc *NodeConfig) Merge(i interface{}) error {
	gc, ok := i.(*NodeConfig)
	if !ok {
		return xerrors.Errorf("failed to merge; invalid type found: %T", i)
	}

	// policy
	if nc.Policy == nil {
		nc.Policy = gc.Policy
	} else if err := merge(nc.Policy, gc.Policy); err != nil {
		return err
	}

	// blocks
	if nc.Blocks == nil {
		nc.Blocks = gc.Blocks
		SortBlocksByHeight(nc.Blocks)
	} else {
		bh := map[string]*BlockConfig{}
		var last *isaac.Height
		for _, b := range nc.Blocks {
			bh[b.Height.String()] = b

			if last == nil || b.Height.Cmp(*last) > 0 {
				last = b.Height
			}
		}

		if gc.Blocks != nil && last.Cmp(*gc.Blocks[len(gc.Blocks)-1].Height) < 0 {
			last = gc.Blocks[len(gc.Blocks)-1].Height
		}

		var blocks []*BlockConfig
		for i := uint64(0); i <= last.Uint64(); i++ {
			height := isaac.NewBlockHeight(i)
			b, found := bh[height.String()]
			if !found {
				if int(i) <= len(gc.Blocks)-1 {
					b = gc.Blocks[int(i)]
				} else {
					round := isaac.Round(0)
					b = &BlockConfig{Height: &height, Round: &round}
				}
			}

			blocks = append(blocks, b)
		}
		nc.Blocks = blocks
		SortBlocksByHeight(nc.Blocks)
	}

	// modules
	if nc.Modules == nil {
		nc.Modules = gc.Modules
	} else {
		if err := merge(nc.Modules, gc.Modules); err != nil {
			return err
		}
	}

	return nil
}

func (nc *NodeConfig) MarshalZerologObject(e *zerolog.Event) {
	e.Interface("Policy", nc.Policy)
	e.Interface("Blocks", nc.Blocks)
	e.Interface("Modules", nc.Modules)
}

type PolicyConfig struct {
	Threshold                         *float64       `yaml:",omitempty"`
	IntervalBroadcastINITBallotInJoin *time.Duration `yaml:"interval_broadcast_init_ballot_in_join,omitempty"`
	TimeoutWaitVoteResultInJoin       *time.Duration `yaml:"timeout_wait_vote_result_in_join,omitempty"`
	TimeoutWaitBallot                 *time.Duration `yaml:"timeout_wait_ballot,omitempty"`
	TimeoutWaitINITBallot             *time.Duration `yaml:"timeout_wait_init_ballot,omitempty"`
}

func defaultPolicyConfig() *PolicyConfig {
	th := float64(67)
	intervalBroadcastINITBallotInJoin := time.Second * 1
	timeoutWaitVoteResultInJoin := time.Second * 3
	timeoutWaitBallot := time.Second * 3
	timeoutWaitINITBallot := time.Second * 3

	return &PolicyConfig{
		Threshold:                         &th,
		IntervalBroadcastINITBallotInJoin: &intervalBroadcastINITBallotInJoin,
		TimeoutWaitVoteResultInJoin:       &timeoutWaitVoteResultInJoin,
		TimeoutWaitBallot:                 &timeoutWaitBallot,
		TimeoutWaitINITBallot:             &timeoutWaitINITBallot,
	}
}

func (pc *PolicyConfig) IsValid() error {
	return nil
}

func (pc *PolicyConfig) Merge(i interface{}) error {
	gc, ok := i.(*PolicyConfig)
	if !ok {
		return xerrors.Errorf("failed to merge; invalid type found: %T", i)
	}

	if pc.Threshold == nil {
		pc.Threshold = gc.Threshold
	}

	if pc.IntervalBroadcastINITBallotInJoin == nil {
		pc.IntervalBroadcastINITBallotInJoin = gc.IntervalBroadcastINITBallotInJoin
	}

	if pc.TimeoutWaitVoteResultInJoin == nil {
		pc.TimeoutWaitVoteResultInJoin = gc.TimeoutWaitVoteResultInJoin
	}

	if pc.TimeoutWaitBallot == nil {
		pc.TimeoutWaitBallot = gc.TimeoutWaitBallot
	}

	if pc.TimeoutWaitINITBallot == nil {
		pc.TimeoutWaitINITBallot = gc.TimeoutWaitINITBallot
	}

	return nil
}

type BlockConfig struct {
	Height *isaac.Height
	Round  *isaac.Round
}

func defaultNBlocksConfig(n int) []*BlockConfig {
	var bs []*BlockConfig

	for i := 0; i < n; i++ {
		height := isaac.NewBlockHeight(uint64(i))
		round := isaac.Round(0)
		bs = append(bs, &BlockConfig{Height: &height, Round: &round})
	}

	return bs
}

func (bc *BlockConfig) IsValid() error {
	if bc.Height == nil {
		return xerrors.Errorf("height is empty")
	}
	if err := bc.Height.IsValid(); err != nil {
		return err
	}

	if bc.Round == nil {
		r := isaac.Round(0)
		bc.Round = &r
	}

	return nil
}

func (bc *BlockConfig) ToBlock() isaac.Block {
	k := fmt.Sprintf("%s-%d", (*bc.Height).String(), (*bc.Round).Uint64())
	r, _ := blockGenerator.LoadOrStore(k, contest_module.NewBlock(*bc.Height, *bc.Round))
	return r.(isaac.Block)
}

type ModulesConfig struct {
	Suffrage      contest_module.SuffrageConfig      `yaml:"suffrage"`
	ProposalMaker contest_module.ProposalMakerConfig `yaml:"proposal_maker"`
	BallotMaker   contest_module.BallotMakerConfig   `yaml:"ballot_maker"`
}

func defaultModulesConfig(numberOfNodes uint) *ModulesConfig {
	return &ModulesConfig{
		Suffrage:      &contest_module.RoundrobinSuffrageConfig{N: "RoundrobinSuffrage", NA: numberOfNodes},
		ProposalMaker: &contest_module.DefaultProposalMakerConfig{N: "DefaultProposalMaker", D: 1},
		BallotMaker:   &contest_module.DefaultBallotMakerConfig{N: "DefaultBallotMaker"},
	}
}

func (mc *ModulesConfig) UnmarshalYAML(value *yaml.Node) error {
	n := struct {
		Suffrage      contest_config.NameBasedConfig `yaml:"suffrage,omitempty"`
		ProposalMaker contest_config.NameBasedConfig `yaml:"proposal_maker,omitempty"`
		BallotMaker   contest_config.NameBasedConfig `yaml:"ballot_maker,omitempty"`
	}{
		Suffrage:      contest_config.NewNameBasedConfig(contest_module.SuffrageConfigs),
		ProposalMaker: contest_config.NewNameBasedConfig(contest_module.ProposalMakerConfigs),
		BallotMaker:   contest_config.NewNameBasedConfig(contest_module.BallotMakerConfigs),
	}

	if err := value.Decode(&n); err != nil {
		return err
	}

	if n.Suffrage.Instance() != nil {
		mc.Suffrage = n.Suffrage.Instance().(contest_module.SuffrageConfig)
	}
	if n.ProposalMaker.Instance() != nil {
		mc.ProposalMaker = n.ProposalMaker.Instance().(contest_module.ProposalMakerConfig)
	}
	if n.BallotMaker.Instance() != nil {
		mc.BallotMaker = n.BallotMaker.Instance().(contest_module.BallotMakerConfig)
	}

	return nil
}

func (mc *ModulesConfig) IsValid() error {
	if mc.Suffrage != nil {
		if err := mc.Suffrage.(contest_config.IsValider).IsValid(); err != nil {
			return err
		}
	}

	if mc.ProposalMaker != nil {
		if err := mc.ProposalMaker.(contest_config.IsValider).IsValid(); err != nil {
			return err
		}
	}

	if mc.BallotMaker != nil {
		if err := mc.BallotMaker.(contest_config.IsValider).IsValid(); err != nil {
			return err
		}
	}

	return nil
}

func (mc *ModulesConfig) Merge(i interface{}) error {
	gc, ok := i.(*ModulesConfig)
	if !ok {
		return xerrors.Errorf("failed to merge; invalid type found: %T", i)
	}

	if mc.Suffrage == nil {
		mc.Suffrage = gc.Suffrage
	} else if err := merge(mc.Suffrage, gc.Suffrage); err != nil {
		return err
	}

	if mc.ProposalMaker == nil {
		mc.ProposalMaker = gc.ProposalMaker
	} else if err := merge(mc.ProposalMaker, gc.ProposalMaker); err != nil {
		return err
	}

	if mc.BallotMaker == nil {
		mc.BallotMaker = gc.BallotMaker
	} else if err := merge(mc.BallotMaker, gc.BallotMaker); err != nil {
		return err
	}

	return nil
}

type ConditionsConfig struct {
	Conditions map[string][]condition.ConditionChecker
}

func (cc *ConditionsConfig) UnmarshalYAML(value *yaml.Node) error {
	m := map[string][]string{}
	if err := value.Decode(&m); err != nil {
		return err
	}

	conditions := map[string][]condition.ConditionChecker{}
	for k, v := range m {
		var l []condition.ConditionChecker
		for _, c := range v {
			cc, err := condition.NewConditionChecker(c)
			if err != nil {
				return err
			}
			l = append(l, cc)
		}

		if len(l) < 1 {
			return xerrors.Errorf("empty conditions in %s", k)
		}
		conditions[k] = l
	}

	cc.Conditions = conditions

	return nil
}

func (cc *ConditionsConfig) MarshalYAML() (interface{}, error) {
	m := map[string][]string{}
	for name, c := range cc.Conditions {
		var l []string
		for _, q := range c {
			l = append(l, q.Query())
		}
		m[name] = l
	}

	return m, nil
}

func (cc *ConditionsConfig) MarshalJSON() ([]byte, error) {
	m, err := cc.MarshalYAML()
	if err != nil {
		return nil, err
	}

	return json.Marshal(m)
}

func (cc *ConditionsConfig) IsValid() error {
	return nil
}

func merge(a, b contest_config.Merger) error {
	if err := a.Merge(b); err != nil {
		return err
	}

	return nil
}

type sortByBlock func(a, b *BlockConfig) bool

func (sortBy sortByBlock) Sort(bs []*BlockConfig) {
	ns := &blocksSorter{
		bs:     bs,
		sortBy: sortBy,
	}
	sort.Sort(ns)
}

type blocksSorter struct {
	bs     []*BlockConfig
	sortBy func(a, b *BlockConfig) bool
}

func (s *blocksSorter) Len() int {
	return len(s.bs)
}

func (s *blocksSorter) Swap(i, j int) {
	s.bs[i], s.bs[j] = s.bs[j], s.bs[i]
}

func (s *blocksSorter) Less(i, j int) bool {
	return s.sortBy(s.bs[i], s.bs[j])
}

func cmpBlocksByHeight(a, b *BlockConfig) bool {
	return (*a.Height).Cmp(*b.Height) < 0
}

func SortBlocksByHeight(bs []*BlockConfig) {
	sortByBlock(cmpBlocksByHeight).Sort(bs)
}
