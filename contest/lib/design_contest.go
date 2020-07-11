package contestlib

import (
	"io/ioutil"
	"path/filepath"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/util/encoder"
)

type ContestDesign struct {
	encs       *encoder.Encoders
	Conditions []*Condition
	Config     *ContestConfigDesign
	Component  *ComponentDesign
	actions    map[string]ConditionActionLoader
	Nodes      []*ContestNodeDesign
}

func LoadContestDesignFromFile(
	f string, encs *encoder.Encoders,
	actions map[string]ConditionActionLoader,
) (*ContestDesign, error) {
	var design ContestDesign
	if b, err := ioutil.ReadFile(filepath.Clean(f)); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal(b, &design); err != nil {
		return nil, err
	}

	design.encs = encs
	design.actions = actions

	return &design, nil
}

func (cd *ContestDesign) IsValid([]byte) error {
	if len(cd.Nodes) < 1 {
		return xerrors.Errorf("empty nodes")
	}

	if err := cd.loadConditionActions(); err != nil {
		return err
	}

	if cd.Config == nil {
		cd.Config = NewContestConfigDesign()
	}
	if err := cd.Config.IsValid(nil); err != nil {
		return err
	}

	if cd.Component == nil {
		cd.Component = NewComponentDesign(nil)
	}
	if err := cd.Component.IsValid(nil); err != nil {
		return err
	}

	for _, n := range cd.Nodes {
		if n.Component == nil {
			n.Component = NewComponentDesign(cd.Component)
		} else if err := n.Component.Merge(cd.Component); err != nil {
			return err
		}

		if err := n.IsValid(nil); err != nil {
			return err
		}
	}

	addrs := map[string]struct{}{}
	for _, r := range cd.Nodes {
		if _, found := addrs[r.Address()]; found {
			return xerrors.Errorf("duplicated address found: '%v'", r.Address())
		}
		addrs[r.Address()] = struct{}{}
	}

	return nil
}

func (cd *ContestDesign) loadConditionActions() error {
	for _, c := range cd.Conditions {
		if err := c.IsValid(nil); err != nil {
			return err
		}

		if len(c.ActionString) < 1 {
			continue
		} else if f, found := cd.actions[c.ActionString]; !found {
			return xerrors.Errorf("action not found: %q", c.ActionString)
		} else {
			ca := NewConditionAction(c.ActionString, f, c.Args, c.IfError)
			if err := ca.IsValid(nil); err != nil {
				return xerrors.Errorf("invalid actions: %w", err)
			}
			c.action = ca
		}
	}

	return nil
}

type ContestConfigDesign struct {
	NetworkIDString string                     `yaml:"network-id"`
	GenesisPolicy   *launcher.PolicyDesign     `yaml:"genesis-policy"`
	InitOperations  []launcher.OperationDesign `yaml:"init-operations"`
}

func NewContestConfigDesign() *ContestConfigDesign {
	return &ContestConfigDesign{
		GenesisPolicy:   launcher.NewPolicyDesign(),
		NetworkIDString: "contest-network-id",
	}
}

func (cc *ContestConfigDesign) IsValid([]byte) error {
	if cc.GenesisPolicy == nil {
		cc.GenesisPolicy = launcher.NewPolicyDesign()
	} else if err := cc.GenesisPolicy.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (cc *ContestConfigDesign) NetworkID() []byte {
	return []byte(cc.NetworkIDString)
}
