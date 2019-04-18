package consensus

import "github.com/spikeekips/mitum/common"

type Consensus interface {
	Name() string
	Start() error
	Stop() error
	Receive(common.Seal) error
}
