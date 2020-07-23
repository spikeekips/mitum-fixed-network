package launcher

import (
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/util/encoder"
	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"github.com/spikeekips/mitum/util/hint"
)

const (
	suffrageTypeRoundrobin    = "roundrobin"
	suffrageTypeFixedProposer = "fixed-proposer"
)

type SuffrageDesign struct {
	Type string
	Info map[string]interface{} `yaml:"-"`
}

func NewSuffrageDesign() *SuffrageDesign {
	return &SuffrageDesign{Type: suffrageTypeRoundrobin, Info: map[string]interface{}{"type": suffrageTypeRoundrobin}}
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
	case suffrageTypeRoundrobin:
	case suffrageTypeFixedProposer:
		if _, found := st.Info["proposer"]; !found {
			return xerrors.Errorf("`proposer` must be set for fixed-suffrage")
		}
	default:
		return xerrors.Errorf("unknown type, %q", st.Type)
	}

	return nil
}

func (st *SuffrageDesign) New(localstate *isaac.Localstate, encs *encoder.Encoders) (base.Suffrage, error) {
	switch st.Type {
	case suffrageTypeRoundrobin:
		return (RoundrobinSuffrageDesign{}).New(localstate, encs)
	case suffrageTypeFixedProposer:
		var proposer string
		if i, found := st.Info["proposer"]; !found {
			return nil, xerrors.Errorf("`proposer` must be set for fixed-suffrage")
		} else {
			proposer = i.(string)
		}

		return (FixedSuffrageDesign{proposer: proposer}).New(localstate, encs)
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

func (fs FixedSuffrageDesign) New(localstate *isaac.Localstate, encs *encoder.Encoders) (base.Suffrage, error) {
	var je encoder.Encoder
	if e, err := encs.Encoder(jsonenc.JSONType, ""); err != nil {
		return nil, xerrors.Errorf("json encoder needs for load design: %w", err)
	} else {
		je = e
	}

	var proposer base.Address
	switch p, err := base.DecodeAddressFromString(je, fs.proposer); {
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

func (fs RoundrobinSuffrageDesign) New(localstate *isaac.Localstate, _ *encoder.Encoders) (base.Suffrage, error) {
	return NewRoundrobinSuffrage(localstate, 100), nil
}

func isValidContestSuffrageDesign(m map[string]interface{}) error {
	var t string
	if i, found := m["type"]; !found {
		return nil
	} else if s, ok := i.(string); !ok {
		return xerrors.Errorf("invalid type value, %T, it should be string", i)
	} else {
		t = s
	}

	switch t {
	case suffrageTypeRoundrobin:
	case suffrageTypeFixedProposer:
		if i, found := m["proposer"]; !found {
			return xerrors.Errorf("`proposer` must be set for fixed-suffrage")
		} else if s, ok := i.(string); !ok {
			return xerrors.Errorf("invalid proposer value, %T, it should be string", i)
		} else if p, err := base.NewStringAddress(s); err != nil {
			return err
		} else {
			m["proposer"] = hint.HintedString(p.Hint(), p.String())
		}
	default:
		return xerrors.Errorf("unknown suffrage type, %q", t)
	}

	return nil
}
