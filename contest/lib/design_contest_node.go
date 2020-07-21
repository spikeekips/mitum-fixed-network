package contestlib

import "github.com/spikeekips/mitum/base"

type ContestNodeDesign struct {
	AddressString string `yaml:"address"`
	Component     *ComponentDesign
}

func (cn *ContestNodeDesign) Address() string {
	return cn.AddressString
}

func (cn *ContestNodeDesign) IsValid([]byte) error {
	if cn.Component == nil {
		cn.Component = NewComponentDesign(nil)
	}

	if err := cn.Component.IsValid(nil); err != nil {
		return err
	}

	if _, err := base.NewStringAddress(cn.AddressString); err != nil {
		return err
	}

	return nil
}
