package main

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v2"

	"github.com/spikeekips/mitum/isaac"
)

type Config struct {
	Global *NodeConfig
	Nodes  map[string]*NodeConfig
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

	if err := cn.Global.IsValid(); err != nil {
		return err
	}

	for _, n := range cn.Nodes {
		if err := n.IsValid(); err != nil {
			return err
		}
	}

	return nil
}

type NodeConfig struct {
	Policy    *PolicyConfig `yaml:",omitempty"`
	LastBlock *BlockConfig  `yaml:"last_block,omitempty"`
}

func defaultNodeConfig() *NodeConfig {
	return &NodeConfig{
		Policy:    defaultPolicyConfig(),
		LastBlock: defaultBlockConfig(),
	}
}

func (nc *NodeConfig) IsValid() error {
	if nc.Policy == nil {
		nc.Policy = defaultPolicyConfig()
	}

	if nc.LastBlock == nil {
		nc.LastBlock = defaultBlockConfig()
	}

	if err := nc.Policy.IsValid(); err != nil {
		return err
	}

	if err := nc.LastBlock.IsValid(); err != nil {
		return err
	}

	return nil
}

type PolicyConfig struct {
	IntervalBroadcastINITBallotInJoin time.Duration `yaml:"interval_broadcast_init_ballot_in_join,omitempty"`
	TimeoutWaitVoteResultInJoin       time.Duration `yaml:"timeout_wait_vote_result_in_join,omitempty"`
	TimeoutWaitBallot                 time.Duration `yaml:"timeout_wait_ballot,omitempty"`
}

func defaultPolicyConfig() *PolicyConfig {
	return &PolicyConfig{
		IntervalBroadcastINITBallotInJoin: time.Second * 1,
		TimeoutWaitVoteResultInJoin:       time.Second * 3,
		TimeoutWaitBallot:                 time.Second * 3,
	}
}

func (pc *PolicyConfig) IsValid() error {
	if pc.IntervalBroadcastINITBallotInJoin < time.Nanosecond {
		log.Warn("IntervalBroadcastINITBallotInJoin is too short", "duration", pc.IntervalBroadcastINITBallotInJoin)
	}

	if pc.TimeoutWaitVoteResultInJoin < time.Nanosecond {
		log.Warn("TimeoutWaitVoteResultInJoin is too short", "duration", pc.TimeoutWaitVoteResultInJoin)
	}

	if pc.TimeoutWaitBallot < time.Nanosecond {
		log.Warn("TimeoutWaitBallot is too short", "duration", pc.TimeoutWaitBallot)
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

func (bc *BlockConfig) IsValid() error {
	if err := bc.Height.IsValid(); err != nil {
		return err
	}

	return nil
}
