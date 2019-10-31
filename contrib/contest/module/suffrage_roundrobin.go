package contest_module

import (
	"encoding/json"

	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
	"golang.org/x/xerrors"
)

func init() {
	Suffrages = append(Suffrages, "RoundrobinSuffrage")
	SuffrageConfigs["RoundrobinSuffrage"] = RoundrobinSuffrageConfig{}
}

type RoundrobinSuffrageConfig struct {
	N  string `yaml:"name"`
	NA uint   `yaml:"number_of_acting"`
}

func (sc RoundrobinSuffrageConfig) Name() string {
	return sc.N
}

func (sc RoundrobinSuffrageConfig) NumberOfActing() uint {
	return sc.NA
}

func (sc *RoundrobinSuffrageConfig) IsValid() error {
	return nil
}

func (sc *RoundrobinSuffrageConfig) Merge(i interface{}) error {
	n, ok := interface{}(i).(SuffrageConfig)
	if !ok {
		return xerrors.Errorf("invalid merge source found: %%", i)
	}

	if sc.NA < 1 {
		sc.NA = n.NumberOfActing()
	}

	return nil
}

func (sc RoundrobinSuffrageConfig) New(homeState *isaac.HomeState, nodes []node.Node, l zerolog.Logger) isaac.Suffrage {
	sf := NewRoundrobinSuffrage(sc.NA, nodes...)
	sf.SetLogger(l)

	return sf
}

type RoundrobinSuffrage struct {
	*common.Logger
	numberOfActing uint // by default numberOfActing is 0; it means all nodes will be acting member
	nodes          []node.Node
}

func NewRoundrobinSuffrage(numberOfActing uint, nodes ...node.Node) *RoundrobinSuffrage {
	if int(numberOfActing) > len(nodes) {
		panic(xerrors.Errorf(
			"numberOfActing should be lesser than number of nodes: numberOfActing=%v nodes=%v",
			numberOfActing,
			len(nodes),
		))
	}

	sorted := make([]node.Node, len(nodes))
	copy(sorted, nodes)

	node.SortNodesByAddress(sorted)

	return &RoundrobinSuffrage{
		Logger: common.NewLogger(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "roundrobin-suffrage")
		}),
		numberOfActing: numberOfActing,
		nodes:          sorted,
	}
}

func (fs *RoundrobinSuffrage) NumberOfActing() uint {
	return fs.numberOfActing
}

func (fs *RoundrobinSuffrage) AddNodes(_ ...node.Node) isaac.Suffrage {
	return fs
}

func (fs *RoundrobinSuffrage) RemoveNodes(_ ...node.Node) isaac.Suffrage {
	return fs
}

func (fs RoundrobinSuffrage) Nodes() []node.Node {
	return fs.nodes
}

func (fs RoundrobinSuffrage) Acting(height isaac.Height, round isaac.Round) isaac.ActingSuffrage {
	nodes := selectNodes(height, round, int(fs.numberOfActing), fs.nodes)

	return isaac.NewActingSuffrage(height, round, nodes[0], nodes)
}

func (fs RoundrobinSuffrage) Exists(_ isaac.Height, address node.Address) bool {
	for _, n := range fs.nodes {
		if n.Address().Equal(address) {
			return true
		}
	}

	return false
}

func (fs RoundrobinSuffrage) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"type":             "RoundrobinSuffrage",
		"nodes":            fs.nodes,
		"number_of_acting": fs.numberOfActing,
	})
}
