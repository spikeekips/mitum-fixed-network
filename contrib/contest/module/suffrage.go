package contest_module

import (
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/node"
)

var Suffrages []string
var SuffrageConfigs map[string]interface{}

func init() {
	SuffrageConfigs = map[string]interface{}{}
}

type SuffrageConfig interface {
	Name() string
	NumberOfActing() uint
	Merge(interface{}) error
	New(*isaac.HomeState, []node.Node, zerolog.Logger) isaac.Suffrage
}
