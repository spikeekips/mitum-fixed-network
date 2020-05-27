package contestlib

type ContestNodeDesign struct {
	AddressString string `yaml:"address"`
	Component     *ContestComponentDesign
}

func (cn *ContestNodeDesign) Address() string {
	return cn.AddressString
}

func (cn *ContestNodeDesign) IsValid([]byte) error {
	if cn.Component == nil {
		cn.Component = NewContestComponentDesign()
	}

	if err := cn.Component.IsValid(nil); err != nil {
		return err
	}

	if _, err := NewContestAddress(cn.AddressString); err != nil {
		return err
	}

	return nil
}
