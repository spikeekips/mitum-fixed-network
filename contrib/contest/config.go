package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v2"

	contest_module "github.com/spikeekips/mitum/contrib/contest/module"
	"github.com/spikeekips/mitum/isaac"
)

type Config struct {
	Global         *NodeConfig
	Nodes          map[string]*NodeConfig
	NumberOfNodes_ *uint `yaml:"number_of_nodes,omitempty"`
}

func LoadConfig(f string, numberOfNodes uint) (*Config, error) {
	log.Debug().
		Uint("number_of_nodes", numberOfNodes).
		Str("file", f).
		Msg("trying to load config")

	b, err := ioutil.ReadFile(filepath.Clean(f))
	if err != nil {
		return nil, xerrors.Errorf("failed to load config: %s, %w", f, err)
	}

	var config Config
	if err := yaml.Unmarshal(b, &config); err != nil {
		return nil, err
	}

	if err := config.IsValid(); err != nil {
		return nil, err
	}

	if numberOfNodes < 1 {
		numberOfNodes = uint(len(config.Nodes))
	}

	// extends nodes by numberOfNodes
	if int(numberOfNodes) > len(config.Nodes) {
		log.Debug().
			Uint("numberOfNodes", numberOfNodes).
			Msg("extend nodes")

		var last int
		for name := range config.Nodes {
			var c int
			if _, err := fmt.Sscanf(name, "n%d", &c); err != nil {
				log.Debug().Str("name", name).Msg("not expected node name format")
				continue
			} else if c > last {
				last = c
			}
		}

		upto := int(numberOfNodes) - len(config.Nodes)
		for i := 0; i < upto; i++ {
			name := fmt.Sprintf("n%d", i+last+1)
			config.Nodes[name] = config.Global
		}
	} else if int(numberOfNodes) < len(config.Nodes) {
		var names []string
		for name := range config.Nodes {
			names = append(names, name)
		}

		sort.Strings(names)
		for _, name := range names[numberOfNodes:] {
			delete(config.Nodes, name)
		}
	}

	config.NumberOfNodes_ = &numberOfNodes

	return &config, nil
}

func (cn *Config) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"global":          cn.Global,
		"nodes":           cn.Nodes,
		"number_of_nodes": cn.NumberOfNodes(),
	})
}

func (cn *Config) MarshalZerologObject(e *zerolog.Event) {
	e.Interface("global", cn.Global)
	e.Interface("nodes", cn.Nodes)
	e.Uint("number_of_nodes", cn.NumberOfNodes())
}

func (cn *Config) String() string {
	b, _ := json.Marshal(cn)
	return string(b)
}

func (cn *Config) IsValid() error {
	if cn.Global == nil {
		cn.Global = defaultNodeConfig()
	}

	if len(cn.Nodes) < 1 {
		log.Warn().Msg("nodes empty")
	}

	if err := cn.Global.IsValid(nil); err != nil {
		return err
	}

	// generate global blocks
	inputs := cn.Global.Blocks[:]
	sort.Slice(
		inputs,
		func(i, j int) bool {
			return (*inputs[i].Height).Cmp(*inputs[j].Height) < 0
		},
	)

	var b isaac.Block
	if inputs[0].Height.Equal(isaac.GenesisHeight) {
		b = NewBlock(*inputs[0].Height, *inputs[0].Round)
		inputs = inputs[1:]
	} else {
		b = NewBlock(isaac.GenesisHeight, isaac.Round(0))
	}

	blocks := map[string]isaac.Block{b.Height().String(): b}
	for _, nextBlock := range inputs {
		if (*nextBlock.Height).Cmp(b.Height()) < 1 {
			return xerrors.Errorf(
				"next height should be greater; previous=%q next=%q",
				b.Height(),
				nextBlock.Height,
			)
		}

		diff := (*nextBlock.Height).Sub(b.Height()).Uint64()
		if diff > 0 {
			for i := uint64(0); i < diff-1; i++ {
				b = NewBlock(b.Height().Add(1), isaac.Round(0))
				blocks[b.Height().String()] = b
			}
		}

		b = NewBlock(*nextBlock.Height, *nextBlock.Round)
		blocks[b.Height().String()] = b
	}

	cn.Global.blocks = blocks

	var nodeNames []string
	for name := range cn.Nodes {
		nodeNames = append(nodeNames, name)
	}
	sort.Strings(nodeNames)

	for i, n := range cn.Nodes {
		if n == nil {
			n = defaultNodeConfig()
			cn.Nodes[i] = n
		}

		if err := n.IsValid(cn.Global); err != nil {
			return err
		}
	}

	return nil
}

func (cn *Config) NumberOfNodes() uint {
	if cn.NumberOfNodes_ == nil {
		return uint(len(cn.Nodes))
	}

	return *cn.NumberOfNodes_
}

type NodeConfig struct {
	Policy  *PolicyConfig  `yaml:",omitempty"`
	Blocks  []*BlockConfig `yaml:"blocks,omitempty"`
	Modules *ModulesConfig `yaml:"modules,omitempty"`
	blocks  map[string]isaac.Block
}

func defaultNodeConfig() *NodeConfig {
	return &NodeConfig{
		Policy: defaultPolicyConfig(),
	}
}

func (nc *NodeConfig) IsValid(global *NodeConfig) error {
	var globalPolicy *PolicyConfig
	var globalBlocks map[string]isaac.Block
	if global != nil {
		globalPolicy = global.Policy
		globalBlocks = global.blocks
	}

	if nc.Policy == nil {
		nc.Policy = globalPolicy
	}

	if err := nc.Policy.IsValid(globalPolicy); err != nil {
		return err
	}

	if nc.Modules == nil {
		if global == nil {
			nc.Modules = defaultModulesConfig()
		} else {
			nc.Modules = global.Modules
		}
	} else {
		var mc *ModulesConfig
		if global != nil {
			mc = global.Modules
		}

		if err := nc.Modules.IsValid(mc); err != nil {
			return err
		}
	}

	if len(nc.Blocks) < 1 {
		nc.blocks = globalBlocks
		nc.Blocks = global.Blocks
	} else if globalBlocks != nil {
		inputs := nc.Blocks[:]
		sort.Slice(
			inputs,
			func(i, j int) bool {
				return (*inputs[i].Height).Cmp(*inputs[j].Height) < 0
			},
		)

		lastBlock := inputs[len(inputs)-1]

		nb := map[string]isaac.Block{}
		for _, b := range globalBlocks {
			if b.Height().Cmp(*lastBlock.Height) > 0 {
				continue
			}
			nb[b.Height().String()] = b
		}
		nc.blocks = nb

		for _, i := range inputs {
			b := NewBlock(*i.Height, *i.Round)
			nc.blocks[b.Height().String()] = b
		}
	}

	return nil
}

func (nc *NodeConfig) Block(height isaac.Height) isaac.Block {
	return nc.blocks[height.String()]
}

func (nc *NodeConfig) LastBlock() isaac.Block {
	height := *nc.Blocks[len(nc.Blocks)-1].Height
	return nc.blocks[height.String()]
}

func (nc *NodeConfig) MarshalZerologObject(e *zerolog.Event) {
	e.Interface("Policy", nc.Policy)
	e.Interface("Blocks", nc.Blocks)
	e.Interface("Modules", nc.Modules)
	e.Interface("blocks", nc.blocks)
}

type PolicyConfig struct {
	Threshold                         *float64       `yaml:",omitempty"`
	IntervalBroadcastINITBallotInJoin *time.Duration `yaml:"interval_broadcast_init_ballot_in_join,omitempty"`
	TimeoutWaitVoteResultInJoin       *time.Duration `yaml:"timeout_wait_vote_result_in_join,omitempty"`
	TimeoutWaitBallot                 *time.Duration `yaml:"timeout_wait_ballot,omitempty"`
}

func defaultPolicyConfig() *PolicyConfig {
	th := float64(67)
	intervalBroadcastINITBallotInJoin := time.Second * 1
	timeoutWaitVoteResultInJoin := time.Second * 3
	timeoutWaitBallot := time.Second * 3

	return &PolicyConfig{
		Threshold:                         &th,
		IntervalBroadcastINITBallotInJoin: &intervalBroadcastINITBallotInJoin,
		TimeoutWaitVoteResultInJoin:       &timeoutWaitVoteResultInJoin,
		TimeoutWaitBallot:                 &timeoutWaitBallot,
	}
}

func (pc *PolicyConfig) IsValid(global *PolicyConfig) error {
	d := defaultPolicyConfig()

	if pc.Threshold == nil {
		pc.Threshold = d.Threshold
	} else if *pc.Threshold < 67 {
		log.Warn().Float64("threshold", *pc.Threshold).Msg("Threshold is too low")
	}

	if *pc.IntervalBroadcastINITBallotInJoin < time.Nanosecond {
		log.Warn().Dur("duration", *pc.IntervalBroadcastINITBallotInJoin).Msg("IntervalBroadcastINITBallotInJoin is too short")
		pc.IntervalBroadcastINITBallotInJoin = global.IntervalBroadcastINITBallotInJoin
	}

	if *pc.TimeoutWaitVoteResultInJoin < time.Nanosecond {
		log.Warn().Dur("duration", *pc.TimeoutWaitVoteResultInJoin).Msg("TimeoutWaitVoteResultInJoin is too short")
		pc.TimeoutWaitVoteResultInJoin = global.TimeoutWaitVoteResultInJoin
	}

	if *pc.TimeoutWaitBallot < time.Nanosecond {
		log.Warn().Dur("duration", *pc.TimeoutWaitBallot).Msg("TimeoutWaitBallot is too short")
		pc.TimeoutWaitBallot = global.TimeoutWaitBallot
	}

	return nil
}

type BlockConfig struct {
	Height *isaac.Height
	Round  *isaac.Round
}

func (bc *BlockConfig) IsValid(*BlockConfig) error {
	if bc.Height == nil {
		return xerrors.Errorf("height is empty")
	}
	if err := bc.Height.IsValid(); err != nil {
		return err
	}

	return nil
}

type ModulesConfig struct {
	Suffrage      *SuffrageConfig `yaml:"suffrage,omitempty"`
	ProposalMaker *ProposalMaker  `yaml:"proposal_maker,omitempty"`
}

func defaultModulesConfig() *ModulesConfig {
	return &ModulesConfig{
		Suffrage: defaultSuffrageConfig(),
	}
}

func (mc *ModulesConfig) IsValid(global *ModulesConfig) error {
	if mc.Suffrage == nil {
		if global == nil {
			mc.Suffrage = defaultSuffrageConfig()
		} else {
			mc.Suffrage = global.Suffrage
		}
	} else {
		var sc *SuffrageConfig
		if global != nil {
			sc = global.Suffrage
		}

		if err := mc.Suffrage.IsValid(sc); err != nil {
			return err
		}
	}

	if mc.ProposalMaker == nil {
		if global == nil {
			mc.ProposalMaker = defaultProposalMaker()
		} else {
			mc.ProposalMaker = global.ProposalMaker
		}
	} else {
		var sc *ProposalMaker
		if global != nil {
			sc = global.ProposalMaker
		}

		if err := mc.ProposalMaker.IsValid(sc); err != nil {
			return err
		}
	}

	return nil
}

type SuffrageConfig map[string]interface{}

func defaultSuffrageConfig() *SuffrageConfig {
	return &SuffrageConfig{
		"name":             "RoundrobinSuffrage",
		"number_of_acting": 0,
	}
}

func (sc *SuffrageConfig) IsValid(global *SuffrageConfig) error {
	if len(*sc) < 1 {
		if global == nil {
			*sc = *defaultSuffrageConfig()
		} else {
			*sc = *global
		}

		return nil
	}

	var found bool
	name := (*sc)["name"]
	for _, n := range contest_module.Suffrages {
		if n == name {
			found = true
			break
		}
	}
	if !found {
		return xerrors.Errorf("unknown suffrage found: %v", name)
	}

	switch name {
	case "FixedProposerSuffrage":
		if _, found := (*sc)["proposer"]; !found {
			return xerrors.Errorf("`proposer` must be given for `FixedProposerSuffrage`")
		}
	case "RoundrobinSuffrage":
		//
	}

	if v, found := (*sc)["number_of_acting"]; !found {
		return xerrors.Errorf("`number_of_acting` must be given")
	} else {
		switch v.(type) {
		case int:
		default:
			return xerrors.Errorf("`number_of_acting` must be int")
		}
	}
	return nil
}

type ProposalMaker map[string]interface{}

func defaultProposalMaker() *ProposalMaker {
	return &ProposalMaker{
		"name":  "DefaultProposalMaker",
		"delay": "1s",
	}
}

func (sc *ProposalMaker) IsValid(global *ProposalMaker) error {
	if len(*sc) < 1 {
		if global == nil {
			*sc = *defaultProposalMaker()
		} else {
			*sc = *global
		}

		return nil
	}

	var found bool
	name := (*sc)["name"]
	for _, n := range contest_module.ProposalMakers {
		if n == name {
			found = true
			break
		}
	}
	if !found {
		return xerrors.Errorf("unknown proposal_maker found: %v", name)
	}

	switch name {
	case "DefaultProposalMaker":
		if s, found := (*sc)["delay"]; !found {
			return xerrors.Errorf("`delay` must be given for `DefaultProposalMaker`")
		} else if d, ok := s.(string); !ok {
			return xerrors.Errorf("`delay` must be time.Duration string format; %v", (*sc)["delay"])
		} else if _, err := time.ParseDuration(d); err != nil {
			return err
		}
	}

	return nil
}
