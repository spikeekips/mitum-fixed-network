package contest_module

import (
	"encoding/json"

	"golang.org/x/xerrors"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

func init() {
	Suffrages = append(Suffrages, "FixedProposerSuffrage")
	SuffrageConfigs["FixedProposerSuffrage"] = FixedProposerSuffrageConfig{}
}

type FixedProposerSuffrageConfig struct {
	N        string `yaml:"name"`
	NA       uint   `yaml:"number_of_acting,omitempty"`
	Proposer string `yaml:"proposer"`
}

func (sc FixedProposerSuffrageConfig) Name() string {
	return sc.N
}

func (sc FixedProposerSuffrageConfig) NumberOfActing() uint {
	return sc.NA
}

func (sc *FixedProposerSuffrageConfig) IsValid() error {
	if len(sc.Proposer) < 1 {
		return xerrors.Errorf("empty `proposer`")
	}

	return nil
}

func (sc *FixedProposerSuffrageConfig) Merge(i interface{}) error {
	n, ok := interface{}(i).(SuffrageConfig)
	if !ok {
		return xerrors.Errorf("invalid merge source found: %%", i)
	}

	if sc.NA < 1 {
		sc.NA = n.NumberOfActing()
	}

	return nil
}

func (sc FixedProposerSuffrageConfig) New(homeState *isaac.HomeState, nodes []node.Node, l zerolog.Logger) isaac.Suffrage {
	var proposer node.Node
	for _, n := range nodes {
		if sc.Proposer == n.Alias() || sc.Proposer == n.Address().String() {
			proposer = n
			break
		}
	}

	if proposer == nil {
		panic(xerrors.Errorf("failed to find proposer: %v", sc.Proposer))
	}

	sf := NewFixedProposerSuffrage(proposer, sc.NumberOfActing(), nodes...)
	sf.SetLogger(l)

	return sf
}

type FixedProposerSuffrage struct {
	*common.Logger
	proposer       node.Node
	numberOfActing uint // by default numberOfActing is 0; it means all nodes will be acting member
	nodes          []node.Node
	others         []node.Node
}

func NewFixedProposerSuffrage(proposer node.Node, numberOfActing uint, nodes ...node.Node) *FixedProposerSuffrage {
	sorted := make([]node.Node, len(nodes))
	copy(sorted, nodes)

	node.SortNodesByAddress(sorted)

	if int(numberOfActing) > len(sorted) {
		panic(xerrors.Errorf(
			"numberOfActing should be lesser than number of nodes: numberOfActing=%v nodes=%v",
			numberOfActing,
			len(sorted),
		))
	}

	var others []node.Node
	for _, n := range sorted {
		if n.Address().Equal(proposer.Address()) {
			continue
		}
		others = append(others, n)
	}

	return &FixedProposerSuffrage{
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "fixed-suffrage")
		}),
		proposer:       proposer,
		numberOfActing: numberOfActing,
		nodes:          sorted,
		others:         others,
	}
}

func (fs FixedProposerSuffrage) NumberOfActing() uint {
	return fs.numberOfActing
}

func (fs FixedProposerSuffrage) AddNodes(_ ...node.Node) isaac.Suffrage {
	return fs
}

func (fs FixedProposerSuffrage) RemoveNodes(_ ...node.Node) isaac.Suffrage {
	return fs
}

func (fs FixedProposerSuffrage) Nodes() []node.Node {
	return fs.nodes
}

func (fs FixedProposerSuffrage) Acting(height isaac.Height, round isaac.Round) isaac.ActingSuffrage {
	var nodes []node.Node
	if fs.numberOfActing == 0 || int(fs.numberOfActing) == len(nodes) {
		nodes = fs.nodes
	} else {
		nodes = append(nodes, fs.proposer)
		nodes = append(
			nodes,
			selectNodes(height, round, int(fs.numberOfActing)-1, fs.others)...,
		)
	}

	return isaac.NewActingSuffrage(height, round, fs.proposer, nodes)
}

func (fs FixedProposerSuffrage) Exists(_ isaac.Height, address node.Address) bool {
	for _, n := range fs.nodes {
		if n.Address().Equal(address) {
			return true
		}
	}

	return false
}

func (fs FixedProposerSuffrage) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":             "FixedProposerSuffrage",
		"proposer":         fs.proposer,
		"nodes":            fs.nodes,
		"number_of_acting": fs.numberOfActing,
	})
}

func selectNodes(height isaac.Height, round isaac.Round, n int, nodes []node.Node) []node.Node {
	if n == 0 || n == len(nodes) {
		return nodes
	}

	var selected []node.Node
	index := (height.Int64() + int64(round)) % int64(len(nodes))
	selected = append(selected, nodes[index:]...)
	if len(selected) < n {
		selected = append(selected, nodes[:n-len(selected)]...)
	} else if len(selected) > n {
		selected = selected[:n]
	}

	return selected
}
