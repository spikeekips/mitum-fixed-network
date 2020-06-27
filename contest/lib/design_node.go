package contestlib

import (
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/util/encoder"
)

type NodeDesign struct {
	*launcher.NodeDesign
	Component *ComponentDesign
}

func LoadNodeDesignFromFile(f string, encs *encoder.Encoders) (*NodeDesign, error) {
	var design *NodeDesign
	if b, err := ioutil.ReadFile(filepath.Clean(f)); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal(b, &design); err != nil {
		return nil, err
	}

	design.SetEncoders(encs)

	return design, nil
}

func (nd NodeDesign) IsEmpty() bool {
	return nd.NodeDesign == nil
}

func (nd *NodeDesign) IsValid([]byte) error {
	if err := nd.NodeDesign.IsValid(nil); err != nil {
		return err
	}

	if nd.Component == nil {
		nd.Component = NewComponentDesign(nil)
	}

	if err := nd.Component.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (nd NodeDesign) MarshalYAML() (interface{}, error) {
	var m map[string]interface{}

	if b, err := yaml.Marshal(nd.NodeDesign); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	m["component"] = nd.Component

	return m, nil
}

func (nd *NodeDesign) UnmarshalYAML(value *yaml.Node) error {
	var d *launcher.NodeDesign
	if err := value.Decode(&d); err != nil {
		return err
	}

	var cp struct {
		Component *ComponentDesign
	}
	if err := value.Decode(&cp); err != nil {
		return err
	}

	*nd = NodeDesign{NodeDesign: d, Component: cp.Component}

	return nil
}
