package launcher

import (
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

type ComponentDesign struct {
	m                 map[string]interface{}
	suffrage          *SuffrageDesign
	proposalProcessor *ProposalProcessorDesign
	others            map[string]interface{}
}

func NewComponentDesign(o *ComponentDesign) *ComponentDesign {
	if o == nil {
		return &ComponentDesign{m: map[string]interface{}{}}
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

	return &ComponentDesign{m: nm}
}

func (cc *ComponentDesign) MarshalYAML() (interface{}, error) {
	return cc.m, nil
}

func (cc *ComponentDesign) UnmarshalYAML(v *yaml.Node) error {
	var m map[string]interface{}
	if err := v.Decode(&m); err != nil {
		return err
	}

	cc.m = m

	return nil
}

func (cc *ComponentDesign) IsValid([]byte) error {
	if cc.m == nil {
		return nil
	}

	var suffrage *SuffrageDesign
	if i, found := cc.m["suffrage"]; !found {
		cc.suffrage = NewSuffrageDesign()
	} else if b, err := yaml.Marshal(i); err != nil {
		return err
	} else if err := yaml.Unmarshal(b, &suffrage); err != nil {
		return err
	} else if err := suffrage.IsValid(nil); err != nil {
		return err
	} else {
		cc.suffrage = suffrage
	}

	var pp *ProposalProcessorDesign
	if i, found := cc.m["proposal-processor"]; !found {
		cc.proposalProcessor = NewProposalProcessorDesign()
	} else if b, err := yaml.Marshal(i); err != nil {
		return err
	} else if err := yaml.Unmarshal(b, &pp); err != nil {
		return err
	} else if err := pp.IsValid(nil); err != nil {
		return err
	} else {
		cc.proposalProcessor = pp
	}

	others := map[string]interface{}{}
	for k, v := range cc.m {
		if k == "suffrage" || k == "proposal-processor" {
			continue
		}

		others[k] = v
	}
	cc.others = others

	return nil
}

func (cc *ComponentDesign) Suffrage() *SuffrageDesign {
	return cc.suffrage
}

func (cc *ComponentDesign) ProposalProcessor() *ProposalProcessorDesign {
	return cc.proposalProcessor
}

func (cc *ComponentDesign) Others() map[string]interface{} {
	return cc.others
}

func (cc *ComponentDesign) M() map[string]interface{} {
	return cc.m
}

func (cc *ComponentDesign) Merge(o *ComponentDesign) error {
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

type ContestComponentDesign struct {
	*ComponentDesign `yaml:",inline"`
}

func NewContestComponentDesign(o *ContestComponentDesign) *ContestComponentDesign {
	var cc *ComponentDesign
	if o == nil {
		cc = NewComponentDesign(nil)
	} else {
		cc = NewComponentDesign(o.ComponentDesign)
	}

	return &ContestComponentDesign{ComponentDesign: cc}
}

func (cc *ContestComponentDesign) UnmarshalYAML(v *yaml.Node) error {
	cd := new(ComponentDesign)
	if err := cd.UnmarshalYAML(v); err != nil {
		return err
	}

	cc.ComponentDesign = cd

	return nil
}

func (cc *ContestComponentDesign) IsValid(b []byte) error {
	if err := cc.ComponentDesign.IsValid(b); err != nil {
		return err
	}

	if i, found := cc.m["suffrage"]; found {
		if m, ok := i.(map[string]interface{}); !ok {
			return xerrors.Errorf("'suffrage' should be map[string]interface{}, not %T", i)
		} else if err := isValidContestSuffrageDesign(m); err != nil {
			return err
		}
	}

	return nil
}

func (cc *ContestComponentDesign) Merge(o *ContestComponentDesign) error {
	return cc.ComponentDesign.Merge(o.ComponentDesign)
}

func (cc *ContestComponentDesign) NodeDesign() *ComponentDesign {
	m, err := DeepCopyMap(cc.m)
	if err != nil {
		panic(err)
	}

	ncc := &ComponentDesign{m: m}
	if err := ncc.IsValid(nil); err != nil {
		panic(err)
	}

	return ncc
}
