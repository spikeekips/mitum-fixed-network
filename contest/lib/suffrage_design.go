package contestlib

import (
	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

type SuffrageComponentDesign struct {
	Type    string
	Info    map[string]interface{} `yaml:"-"`
	creator SuffrageDesign
}

func NewSuffrageComponentDesign() *SuffrageComponentDesign {
	return &SuffrageComponentDesign{Type: "roundrobin", Info: map[string]interface{}{"type": "roundrobin"}}
}

func (st *SuffrageComponentDesign) MarshalYAML() (interface{}, error) {
	return st.Info, nil
}

func (st *SuffrageComponentDesign) UnmarshalYAML(value *yaml.Node) error {
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

func (st *SuffrageComponentDesign) IsValid([]byte) error {
	if st == nil {
		return nil
	}

	switch st.Type {
	case "roundrobin":
		st.creator = RoundrobinSuffrageDesign{}
	case "fixed-proposer":
		var proposer string
		if i, found := st.Info["proposer"]; !found {
			return xerrors.Errorf("`proposer` must be set for fixed-suffrage")
		} else {
			proposer = i.(string)
		}

		st.creator = FixedSuffrageDesign{proposer: proposer}
	default:
		return xerrors.Errorf("unknown type, %q", st.Type)
	}

	return nil
}

func (st *SuffrageComponentDesign) New(localstate *isaac.Localstate) (base.Suffrage, error) {
	if st == nil || st.creator == nil {
		return (RoundrobinSuffrageDesign{}).New(localstate)
	}

	return st.creator.New(localstate)
}

type SuffrageDesign interface {
	New(*isaac.Localstate) (base.Suffrage, error)
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
	var proposer ContestAddress
	switch p, err := NewContestAddress(fs.proposer); {
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

	return base.NewFixedSuffrage(proposer, nodes), nil
}

type RoundrobinSuffrageDesign struct {
}

func (fs RoundrobinSuffrageDesign) IsValid([]byte) error {
	return nil
}

func (fs RoundrobinSuffrageDesign) New(localstate *isaac.Localstate) (base.Suffrage, error) {
	return NewRoundrobinSuffrage(localstate, 100), nil
}
