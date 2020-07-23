package launcher

import (
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

type ContestComponentDesign struct {
	m map[string]interface{}
}

func NewContestComponentDesign(o *ContestComponentDesign) *ContestComponentDesign {
	if o == nil {
		return &ContestComponentDesign{m: map[string]interface{}{}}
	}

	var m map[string]interface{}
	if i, err := DeepCopyMap(o.m); err != nil {
		panic(err)
	} else {
		m = i
	}

	nm := map[string]interface{}{}
	for k := range m {
		nm[k] = m[k]
	}

	return &ContestComponentDesign{m: nm}
}

func (cc *ContestComponentDesign) UnmarshalYAML(v *yaml.Node) error {
	var m map[string]interface{}
	if err := v.Decode(&m); err != nil {
		return err
	}

	cc.m = m

	return nil
}

func (cc *ContestComponentDesign) IsValid([]byte) error {
	if cc.m == nil {
		return nil
	}

	if i, found := cc.m["suffrage"]; found {
		if m, ok := i.(map[string]interface{}); !ok {
			return xerrors.Errorf("'suffrage' should be map[string]interface{}, not %T", i)
		} else if err := isValidContestSuffrageDesign(m); err != nil {
			return err
		}
	}

	var nc *NodeComponentDesign
	if b, err := yaml.Marshal(cc.m); err != nil {
		return err
	} else if err := yaml.Unmarshal(b, &nc); err != nil {
		return err
	} else if err := nc.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (cc *ContestComponentDesign) Merge(o *ContestComponentDesign) error {
	var m map[string]interface{}
	if i, err := DeepCopyMap(o.m); err != nil {
		return err
	} else {
		m = i
	}

	for k := range m {
		if _, found := cc.m[k]; found {
			continue
		}

		cc.m[k] = m[k]
	}

	return cc.IsValid(nil)
}

func (cc *ContestComponentDesign) NodeDesign() *NodeComponentDesign {
	var nc *NodeComponentDesign
	if b, err := yaml.Marshal(cc.m); err != nil {
		panic(err)
	} else if err := yaml.Unmarshal(b, &nc); err != nil {
		panic(err)
	} else if err := nc.IsValid(nil); err != nil {
		panic(err)
	}

	return nc
}

type NodeComponentDesign struct {
	Suffrage          *SuffrageDesign          `yaml:",omitempty"`
	ProposalProcessor *ProposalProcessorDesign `yaml:"proposal-processor,omitempty"`
}

func NewNodeComponentDesign(o *NodeComponentDesign) *NodeComponentDesign {
	if o != nil {
		return &NodeComponentDesign{
			Suffrage:          o.Suffrage,
			ProposalProcessor: o.ProposalProcessor,
		}
	}

	return &NodeComponentDesign{}
}

func (cc *NodeComponentDesign) IsValid([]byte) error {
	if cc.Suffrage == nil {
		cc.Suffrage = NewSuffrageDesign()
	}

	if err := cc.Suffrage.IsValid(nil); err != nil {
		return err
	}

	if cc.ProposalProcessor == nil {
		cc.ProposalProcessor = NewProposalProcessorDesign()
	}

	if err := cc.ProposalProcessor.IsValid(nil); err != nil {
		return err
	}

	return nil
}

func (cc *NodeComponentDesign) Merge(b *NodeComponentDesign) error {
	if cc.Suffrage == nil {
		if b == nil {
			cc.Suffrage = NewSuffrageDesign()
		} else {
			cc.Suffrage = b.Suffrage
		}
	}

	if cc.ProposalProcessor == nil {
		if b == nil {
			cc.ProposalProcessor = NewProposalProcessorDesign()
		} else {
			cc.ProposalProcessor = b.ProposalProcessor
		}
	}

	return nil
}
