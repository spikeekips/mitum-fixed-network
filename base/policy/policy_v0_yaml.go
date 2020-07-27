package policy

import (
	"github.com/spikeekips/mitum/base"
	"gopkg.in/yaml.v3"
)

type PolicyV0PackerYAML struct {
	TR base.ThresholdRatio `yaml:"threshold"`
	NS uint                `yaml:"number-of-acting-suffrage-nodes"`
	MS uint                `yaml:"max-operations-in-seal"`
	MP uint                `yaml:"max-operations-in-proposal"`
}

func (po PolicyV0) MarshalYAML() (interface{}, error) {
	return PolicyV0PackerYAML{
		TR: po.thresholdRatio,
		NS: po.numberOfActingSuffrageNodes,
		MS: po.maxOperationsInSeal,
		MP: po.maxOperationsInProposal,
	}, nil
}

func (po *PolicyV0) UnmarshalYAML(v *yaml.Node) error {
	var up PolicyV0PackerYAML
	if err := v.Decode(&up); err != nil {
		return err
	}

	return po.unpack(up.TR, up.NS, up.MS, up.MP)
}
