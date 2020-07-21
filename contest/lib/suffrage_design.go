package contestlib

import (
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launcher"
	"github.com/spikeekips/mitum/network"
)

type SuffrageDesign struct {
	Type string
	Info map[string]interface{} `yaml:"-"`
}

func NewSuffrageDesign() *SuffrageDesign {
	return &SuffrageDesign{Type: "roundrobin", Info: map[string]interface{}{"type": "roundrobin"}}
}

func (st *SuffrageDesign) MarshalYAML() (interface{}, error) {
	return st.Info, nil
}

func (st *SuffrageDesign) UnmarshalYAML(value *yaml.Node) error {
	var m map[string]interface{}
	if err := value.Decode(&m); err != nil {
		return err
	}

	if t, found := m["type"]; !found {
		return xerrors.Errorf("`type` must be set in suffrage")
	} else {
		st.Type = t.(string)
	}

	st.Info = m

	return nil
}

func (st *SuffrageDesign) IsValid([]byte) error {
	switch st.Type {
	case "roundrobin":
	case "fixed-proposer":
		if _, found := st.Info["proposer"]; !found {
			return xerrors.Errorf("`proposer` must be set for fixed-suffrage")
		}
	default:
		return xerrors.Errorf("unknown type, %q", st.Type)
	}

	return nil
}

func (st *SuffrageDesign) New(localstate *isaac.Localstate) (base.Suffrage, error) {
	switch st.Type {
	case "roundrobin":
		return (RoundrobinSuffrageDesign{}).New(localstate)
	case "fixed-proposer":
		var proposer string
		if i, found := st.Info["proposer"]; !found {
			return nil, xerrors.Errorf("`proposer` must be set for fixed-suffrage")
		} else {
			proposer = i.(string)
		}

		return (FixedSuffrageDesign{proposer: proposer}).New(localstate)
	default:
		return nil, xerrors.Errorf("unknown type found: %v", st.Type)
	}
}

type FixedSuffrageDesign struct {
	proposer string
}

func (fs FixedSuffrageDesign) IsValid([]byte) error {
	if len(fs.proposer) < 1 {
		return xerrors.Errorf("proposer must be set for fixed-suffrage")
	}

	return nil
}

func (fs FixedSuffrageDesign) New(localstate *isaac.Localstate) (base.Suffrage, error) {
	var proposer base.StringAddress
	switch p, err := base.NewStringAddress(fs.proposer); {
	case err != nil:
		return nil, err
	case !p.Equal(localstate.Node().Address()) && !localstate.Nodes().Exists(p):
		return nil, xerrors.Errorf("proposer, %q not found", p)
	default:
		proposer = p
	}

	nodes := make([]base.Address, localstate.Nodes().Len()+1)

	var i int
	localstate.Nodes().Traverse(func(n network.Node) bool {
		nodes[i] = n.Address()

		i++

		return true
	})
	nodes[i] = localstate.Node().Address()

	sf := base.NewFixedSuffrage(proposer, nodes)

	return sf, sf.Initialize()
}

type RoundrobinSuffrageDesign struct {
}

func (fs RoundrobinSuffrageDesign) IsValid([]byte) error {
	return nil
}

func (fs RoundrobinSuffrageDesign) New(localstate *isaac.Localstate) (base.Suffrage, error) {
	return launcher.NewRoundrobinSuffrage(localstate, 100), nil
}
