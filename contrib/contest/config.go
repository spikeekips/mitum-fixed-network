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
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/contrib/contest/condition"
	contest_module "github.com/spikeekips/mitum/contrib/contest/module"
	"github.com/spikeekips/mitum/isaac"
)

type Config struct {
	Global         *NodeConfig
	Nodes          map[string]*NodeConfig
	NumberOfNodes_ *uint `yaml:"number_of_nodes,omitempty"`
	Condition      map[string]*ConditionConfig
}

func LoadConfig(f string, numberOfNodes uint) (*Config, error) {
	log.Debug().
		Uint("number_of_nodes", numberOfNodes).
		Str("file", f).
		Msg("trying to load config")

	b, err := ioutil.ReadFile(filepath.Clean(f))
	if err != nil {
		return nil, xerrors.Errorf("failed to load config(%s): %w", f, err)
	}

	var config Config
	if err := yaml.Unmarshal(b, &config); err != nil {
		return nil, err
	}

	if err := config.IsValid(); err != nil {
		return nil, err
	}

	var last uint
	for name := range config.Nodes {
		var c uint
		if _, err := fmt.Sscanf(name, "n%d", &c); err != nil {
			err := xerrors.Errorf("unexpected node name format: node name should be `n<digit>`")
			log.Error().Err(err).Str("name", name).Send()
			return nil, err
		} else if c > last {
			last = c
		}
	}

	if last < 1 {
		return nil, xerrors.Errorf("number-of-nodes should be greater than 0")
	} else {
		numberOfNodes = last + 1
	}
	config.NumberOfNodes_ = &numberOfNodes

	// check node names
	var nodeNames []string
	for name := range config.Nodes {
		var c uint
		if _, err := fmt.Sscanf(name, "n%d", &c); err != nil {
			err := xerrors.Errorf("unexpected node name format: node name should be `n<digit>`")
			log.Error().Err(err).Str("name", name).Send()
			return nil, err
		}

		nodeNames = append(nodeNames, name)
	}

	sort.Slice(
		nodeNames,
		func(i, j int) bool {
			var ni, nj int
			_, _ = fmt.Sscanf(nodeNames[i], "n%d", &ni)
			_, _ = fmt.Sscanf(nodeNames[j], "n%d", &nj)
			return ni < nj
		},
	)

	nodes := map[string]*NodeConfig{}
	for i := 0; i < int(numberOfNodes); i++ {
		name := fmt.Sprintf("n%d", i)
		n, found := config.Nodes[name]
		if !found {
			n = config.Global
		}
		nodes[name] = n
	}

	config.Nodes = nodes

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

	if cn.Nodes == nil {
		cn.Nodes = map[string]*NodeConfig{}
	}

	if len(cn.Nodes) < 1 {
		log.Warn().Msg("nodes empty")
	}

	if err := cn.Global.IsValid(nil); err != nil {
		return err
	}

	// default blocks for no blocks found in Global config.
	if len(cn.Global.Blocks) < 1 {
		cn.Global.Blocks = []*BlockConfig{
			NewBlockConfig(isaac.NewBlockHeight(9), isaac.Round(10)),
			NewBlockConfig(isaac.NewBlockHeight(10), isaac.Round(11)),
		}
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
	if len(inputs) > 0 && inputs[0].Height.Equal(isaac.GenesisHeight) {
		b = contest_module.NewBlock(*inputs[0].Height, *inputs[0].Round)
		inputs = inputs[1:]
	} else {
		b = contest_module.NewBlock(isaac.GenesisHeight, isaac.Round(0))
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
				b = contest_module.NewBlock(b.Height().Add(1), isaac.Round(0))
				blocks[b.Height().String()] = b
			}
		}

		b = contest_module.NewBlock(*nextBlock.Height, *nextBlock.Round)
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

	all, found := cn.Condition["all"]
	if !found {
		all = defaultConditionConfig()
	}

	for _, c := range cn.Condition {
		if err := c.IsValid(all); err != nil {
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
	if global == nil {
		global = defaultNodeConfig()
	}

	if nc.Policy == nil {
		nc.Policy = global.Policy
	}

	if err := nc.Policy.IsValid(global.Policy); err != nil {
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

	if len(nc.Blocks) < 1 && global.Blocks != nil {
		nc.blocks = global.blocks
		nc.Blocks = global.Blocks
	} else if global.Blocks != nil {
		inputs := nc.Blocks[:]
		sort.Slice(
			inputs,
			func(i, j int) bool {
				return (*inputs[i].Height).Cmp(*inputs[j].Height) < 0
			},
		)

		lastBlock := inputs[len(inputs)-1]

		nb := map[string]isaac.Block{}
		for _, b := range global.blocks {
			if b.Height().Cmp(*lastBlock.Height) > 0 {
				continue
			}
			nb[b.Height().String()] = b
		}
		nc.blocks = nb

		for _, i := range inputs {
			b := contest_module.NewBlock(*i.Height, *i.Round)
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

	if *pc.TimeoutWaitINITBallot < time.Nanosecond {
		log.Warn().Dur("duration", *pc.TimeoutWaitINITBallot).Msg("TimeoutWaitINITBallot is too short")
		pc.TimeoutWaitINITBallot = global.TimeoutWaitINITBallot
	}

	return nil
}

type BlockConfig struct {
	Height *isaac.Height
	Round  *isaac.Round
}

func NewBlockConfig(height isaac.Height, round isaac.Round) *BlockConfig {
	return &BlockConfig{Height: &height, Round: &round}
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
	Suffrage      *SuffrageConfig      `yaml:"suffrage,omitempty"`
	ProposalMaker *ProposalMakerConfig `yaml:"proposal_maker,omitempty"`
	BallotMaker   *BallotMakerConfig   `yaml:"ballot_maker,omitempty"`
}

func defaultModulesConfig() *ModulesConfig {
	return &ModulesConfig{
		Suffrage:      defaultSuffrageConfig(),
		ProposalMaker: defaultProposalMakerConfig(),
		BallotMaker:   defaultBallotMakerConfig(),
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
			mc.ProposalMaker = defaultProposalMakerConfig()
		} else {
			mc.ProposalMaker = global.ProposalMaker
		}
	} else {
		var sc *ProposalMakerConfig
		if global != nil {
			sc = global.ProposalMaker
		}

		if err := mc.ProposalMaker.IsValid(sc); err != nil {
			return err
		}
	}

	if mc.BallotMaker == nil {
		if global == nil {
			mc.BallotMaker = defaultBallotMakerConfig()
		} else {
			mc.BallotMaker = global.BallotMaker
		}
	} else {
		var sc *BallotMakerConfig
		if global != nil {
			sc = global.BallotMaker
		}

		if err := mc.BallotMaker.IsValid(sc); err != nil {
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
		log.Warn().Msg("number_of_acting is missing; the total number of nodes will be number_of_acting")
		(*sc)["number_of_acting"] = 0
	} else {
		switch v.(type) {
		case int:
		default:
			return xerrors.Errorf("`number_of_acting` must be int")
		}
	}
	return nil
}

type ProposalMakerConfig map[string]interface{}

func defaultProposalMakerConfig() *ProposalMakerConfig {
	return &ProposalMakerConfig{
		"name":  "DefaultProposalMaker",
		"delay": "1s",
	}
}

func (sc *ProposalMakerConfig) IsValid(global *ProposalMakerConfig) error {
	if len(*sc) < 1 {
		if global == nil {
			*sc = *defaultProposalMakerConfig()
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

type BallotMakerConfig map[string]interface{}

func defaultBallotMakerConfig() *BallotMakerConfig {
	return &BallotMakerConfig{
		"name": "DefaultBallotMaker",
	}
}

func (bmc *BallotMakerConfig) IsValid(global *BallotMakerConfig) error {
	if len(*bmc) < 1 {
		if global == nil {
			*bmc = *defaultBallotMakerConfig()
		} else {
			*bmc = *global
		}

		return nil
	}

	var found bool
	name := (*bmc)["name"]
	for _, n := range contest_module.BallotMakers {
		if n == name {
			found = true
			break
		}
	}
	if !found {
		return xerrors.Errorf("unknown ballot_maker found: %v", name)
	}

	switch name {
	case "DefaultBallotMaker":
	case "DamangedBallotMaker": // NOTE Deprecated
		// height
		if s, found := (*bmc)["height"]; !found {
			//
		} else if d, ok := s.(int); !ok || d < 0 {
			return xerrors.Errorf("`height` must be uint; %v", (*bmc)["height"])
		}

		// round
		if s, found := (*bmc)["round"]; !found {
			//
		} else if d, ok := s.(int); !ok || d < 0 {
			return xerrors.Errorf("`round` must be uint; %v", (*bmc)["round"])
		}

		// stage
		if s, found := (*bmc)["stage"]; !found {
			//
		} else if d, ok := s.(string); !ok {
			return xerrors.Errorf("`stage` must be string; %v", (*bmc)["stage"])
		} else if _, err := isaac.StageFromString(d); err != nil {
			return xerrors.Errorf("`round` must be valid Stage; %v: %w", (*bmc)["stage"], err)
		}
	case "ConditionBallotMaker":
		if s, found := (*bmc)["conditions"]; !found {
			log.Warn().Msg("conditions is missing")
		} else {
			for _, c := range s.(BallotMakerConfig) {
				if _, err := parseConditionHandler(c.(BallotMakerConfig)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

type ConditionConfig map[string][]string

func defaultConditionConfig() *ConditionConfig {
	return &ConditionConfig{}
}

func (cc *ConditionConfig) IsValid(global *ConditionConfig) error {
	if len(*cc) < 1 {
		if global == nil {
			*cc = *defaultConditionConfig()
		} else {
			*cc = *global
		}

		return nil
	}

	cp := condition.NewConditionParser()
	for _, qs := range *cc {
		for _, q := range qs {
			var err error
			if _, e := cp.Parse(q); e != nil {
				err = xerrors.Errorf("invalid query found; %q: %w", q, e)
			}
			log.Debug().
				Err(err).
				Str("query", q).
				Msg("condition query parsed")

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func parseConditionHandler(m map[string]interface{}) (contest_module.ConditionBallotHandler, error) {
	var query, action string
	if s, found := m["condition"]; !found {
		err := xerrors.Errorf("condition is missing in condition block")
		log.Error().Err(err).Send()
		return contest_module.ConditionBallotHandler{}, err
	} else {
		query = s.(string)
	}

	if s, found := m["action"]; !found {
		err := xerrors.Errorf("action is missing in condition block")
		log.Error().Err(err).Send()
		return contest_module.ConditionBallotHandler{}, err
	} else {
		action = s.(string)
	}

	cc, err := condition.NewConditionChecker(query)
	if err != nil {
		log.Error().Err(err).Send()
		return contest_module.ConditionBallotHandler{}, err
	}

	return contest_module.NewConditionBallotHandler(cc, action), nil
}
