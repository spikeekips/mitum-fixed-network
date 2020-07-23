package contestlib

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/launcher"
)

type ContestNodeDesign struct {
	Name      string
	Component *launcher.ContestComponentDesign
	address   base.Address
}

func (cn *ContestNodeDesign) Address() base.Address {
	return cn.address
}

func (cn *ContestNodeDesign) IsValid([]byte) error {
	if cn.Component == nil {
		cn.Component = launcher.NewContestComponentDesign(nil)
	}

	if err := cn.Component.IsValid(nil); err != nil {
		return err
	}

	if a, err := base.NewStringAddress(cn.Name); err != nil {
		return err
	} else {
		cn.address = a
	}

	return nil
}
