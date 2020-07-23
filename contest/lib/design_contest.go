package contestlib

import (
	"io/ioutil"
	"path/filepath"
	"strings"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/util/encoder"
)

var defaultNetworkID = "contest-network-id"

type ContestDesign struct {
	encs       *encoder.Encoders
	Vars       *Vars
	Conditions []*Condition
	Config     *ContestConfigDesign
	Component  *launcher.ContestComponentDesign
	actions    map[string]ConditionActionLoader
	Nodes      []*ContestNodeDesign
}

func LoadContestDesignFromFile(
	f string,
	encs *encoder.Encoders,
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
		cd.Config = &ContestConfigDesign{}
	}
	if err := cd.Config.IsValid(nil); err != nil {
		return err
	}

	if cd.Component == nil {
		cd.Component = launcher.NewContestComponentDesign(nil)
	}

	for _, n := range cd.Nodes {
		if n.Component == nil {
			n.Component = launcher.NewContestComponentDesign(cd.Component)
		}

		if err := n.Component.Merge(cd.Component); err != nil {
			return err
		}
	}

	if err := cd.Component.IsValid(nil); err != nil {
		return err
	}

	addrs := map[string]struct{}{}
	for _, r := range cd.Nodes {
		if _, found := addrs[r.Name]; found {
			return xerrors.Errorf("duplicated address found: '%v'", r.Name)
		}
		addrs[r.Name] = struct{}{}
	}

	if cd.Vars == nil {
		cd.Vars = NewVars(nil)
	}

	return nil
}

func (cd *ContestDesign) loadConditionActions() error {
	for _, c := range cd.Conditions {
		if err := c.IsValid(nil); err != nil {
			return err
		}

		if err := cd.loadConditionAction(c); err != nil {
			return err
		}
	}

	return nil
}

func (cd *ContestDesign) loadConditionAction(c *Condition) error {
	if len(c.ActionString) < 1 {
		return nil
	}

	var cl ConditionActionLoader
	var args []string
	if strings.HasPrefix(c.ActionString, "$ ") {
		if i, err := NewShellConditionActionLoader(cd.Vars, cutShellCommandString(c.ActionString)); err != nil {
			return err
		} else {
			cl = i
		}
	} else if f, found := cd.actions[c.ActionString]; !found {
		return xerrors.Errorf("action not found: %q", c.ActionString)
	} else {
		cl = f
		args = c.Args
	}

	ca := NewConditionAction(c.ActionString, cl, args, c.IfError)
	if err := ca.IsValid(nil); err != nil {
		return xerrors.Errorf("invalid actions: %w", err)
	}
	c.action = ca

	return nil
}

type ContestConfigDesign struct {
	NetworkIDString string                     `yaml:"network-id"`
	GenesisPolicy   *launcher.PolicyDesign     `yaml:"genesis-policy"`
	InitOperations  []launcher.OperationDesign `yaml:"init-operations"`
	NodeINITCommand string                     `yaml:"node-init-command"`
	NodeRunCommand  string                     `yaml:"node-run-command"`
}

func (cc *ContestConfigDesign) IsValid([]byte) error {
	if cc.GenesisPolicy == nil {
		cc.GenesisPolicy = launcher.NewPolicyDesign()
	}
	if err := cc.GenesisPolicy.IsValid(nil); err != nil {
		return err
	}

	if len(strings.TrimSpace(cc.NetworkIDString)) < 1 {
		cc.NetworkIDString = defaultNetworkID
	}

	cc.NodeINITCommand = strings.TrimSpace(cc.NodeINITCommand)
	cc.NodeRunCommand = strings.TrimSpace(cc.NodeRunCommand)

	return nil
}

func (cc *ContestConfigDesign) NetworkID() []byte {
	return []byte(cc.NetworkIDString)
}
