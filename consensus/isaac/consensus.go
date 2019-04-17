package isaac

import "github.com/spikeekips/mitum/common"

type Consensus struct {
}

func NewConsensus() (*Consensus, error) {
	return &Consensus{}, nil
}

func (c *Consensus) Name() string {
	return "isaac"
}

func (c *Consensus) Start() error {
	return nil
}

func (c *Consensus) Stop() error {
	return nil
}

func (c *Consensus) Receive(s common.Seal) error {
	return nil
}
