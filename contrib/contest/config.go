package main

import (
	"encoding/json"
	"io/ioutil"
	"sort"
	"time"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v2"

	"github.com/spikeekips/mitum/isaac"
)

type Config struct {
	Global         *NodeConfig
	Nodes          map[string]*NodeConfig
	NumberOfNodes_ *uint `yaml:"number_of_nodes,omitempty"`
}

func LoadConfig(f string) (*Config, error) {
	log.Debug("trying to load config", "file", f)
	b, err := ioutil.ReadFile(f)
	if err != nil {
		return nil, xerrors.Errorf("failed to load config: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(b, &config); err != nil {
		return nil, err
	}

	log.Debug("config loaded", "config", config.String())

	if err := config.IsValid(); err != nil {
		return nil, err
	}

	return &config, nil
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
		log.Warn("nodes empty")
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
				b = isaac.NewRandomNextBlock(b)
				blocks[b.Height().String()] = b
			}
		}

		b = NewBlock(*nextBlock.Height, *nextBlock.Round)
		blocks[b.Height().String()] = b

	}

	cn.Global.blocks = blocks

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
	if cn.NumberOfNodes_ != nil {
		return uint(len(cn.Nodes))
	}

	return *cn.NumberOfNodes_
}

type NodeConfig struct {
	Policy *PolicyConfig  `yaml:",omitempty"`
	Blocks []*BlockConfig `yaml:"blocks,omitempty"`
	blocks map[string]isaac.Block
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

	if len(nc.Blocks) < 1 {
		nc.blocks = globalBlocks
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

type PolicyConfig struct {
	Threshold                         *uint          `yaml:",omitempty"`
	IntervalBroadcastINITBallotInJoin *time.Duration `yaml:"interval_broadcast_init_ballot_in_join,omitempty"`
	TimeoutWaitVoteResultInJoin       *time.Duration `yaml:"timeout_wait_vote_result_in_join,omitempty"`
	TimeoutWaitBallot                 *time.Duration `yaml:"timeout_wait_ballot,omitempty"`
}

func defaultPolicyConfig() *PolicyConfig {
	th := uint(67)
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
		log.Warn("Threshold is too low", "threshold", pc.Threshold)
	}

	if *pc.IntervalBroadcastINITBallotInJoin < time.Nanosecond {
		log.Warn("IntervalBroadcastINITBallotInJoin is too short", "duration", pc.IntervalBroadcastINITBallotInJoin)
		pc.IntervalBroadcastINITBallotInJoin = global.IntervalBroadcastINITBallotInJoin
	}

	if *pc.TimeoutWaitVoteResultInJoin < time.Nanosecond {
		log.Warn("TimeoutWaitVoteResultInJoin is too short", "duration", pc.TimeoutWaitVoteResultInJoin)
		pc.TimeoutWaitVoteResultInJoin = global.TimeoutWaitVoteResultInJoin
	}

	if *pc.TimeoutWaitBallot < time.Nanosecond {
		log.Warn("TimeoutWaitBallot is too short", "duration", pc.TimeoutWaitBallot)
		pc.TimeoutWaitBallot = global.TimeoutWaitBallot
	}

	return nil
}

type BlockConfig struct {
	Height *isaac.Height
	Round  *isaac.Round
}

func defaultBlockConfig() *BlockConfig {
	height := isaac.NewBlockHeight(33)
	round := isaac.Round(0)

	return &BlockConfig{
		Height: &height,
		Round:  &round,
	}
}

func (bc *BlockConfig) IsValid(global *BlockConfig) error {
	if bc.Height == nil {
		return xerrors.Errorf("height is empty")
	}
	if err := bc.Height.IsValid(); err != nil {
		return err
	}

	return nil
}
