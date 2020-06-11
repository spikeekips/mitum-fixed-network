package contestlib

import (
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
)

type ProposalProcessorDesign struct {
	Type               string
	Info               map[string]interface{} `yaml:"-"`
	errorINITTHeights  []base.Height
	errorACCEPTHeights []base.Height
}

func NewProposalProcessorDesign() *ProposalProcessorDesign {
	return &ProposalProcessorDesign{Type: "default", Info: map[string]interface{}{
		"type": "default",
	}}
}

func (st *ProposalProcessorDesign) MarshalYAML() (interface{}, error) {
	return st.Info, nil
}

func (st *ProposalProcessorDesign) UnmarshalYAML(value *yaml.Node) error {
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

func (st *ProposalProcessorDesign) IsValid([]byte) error {
	if st == nil {
		return nil
	}

	switch st.Type {
	case "default":
	case "error-when-height":
		var initHeights, acceptHeights []base.Height
		if i, found := st.Info["init-heights"]; !found {
		} else if hs, err := st.parseHeights(i); err != nil {
			return xerrors.Errorf("invalid heights for init error heights: %w", err)
		} else {
			initHeights = hs
		}
		if i, found := st.Info["accept-heights"]; !found {
		} else if hs, err := st.parseHeights(i); err != nil {
			return xerrors.Errorf("invalid heights for accept error heights: %w", err)
		} else {
			acceptHeights = hs
		}

		if len(initHeights) < 1 && len(acceptHeights) < 1 {
			return xerrors.Errorf("accept or init heights must be set for error-when-height")
		}

		st.errorINITTHeights = initHeights
		st.errorACCEPTHeights = acceptHeights
	default:
		return xerrors.Errorf("unknown type, %q", st.Type)
	}

	return nil
}

func (st *ProposalProcessorDesign) New(
	localstate *isaac.Localstate, suffrage base.Suffrage,
) (isaac.ProposalProcessor, error) {
	switch st.Type {
	case "default":
		return isaac.NewProposalProcessorV0(localstate, suffrage), nil
	case "error-when-height":
		return NewErrorProposalProcessor(localstate, suffrage, st.errorINITTHeights, st.errorACCEPTHeights), nil
	default:
		return nil, xerrors.Errorf("unknown type found: %v", st.Type)
	}
}

func (st *ProposalProcessorDesign) parseHeights(hs interface{}) ([]base.Height, error) {
	l, ok := hs.([]interface{})
	if !ok {
		return nil, xerrors.Errorf("`heights` must be list; %T", hs)
	}
	heights := make([]base.Height, len(l))

	for i, v := range l {
		var j int64
		switch t := v.(type) {
		case int:
			j = int64(t)
		case int8:
			j = int64(t)
		case int16:
			j = int64(t)
		case int32:
			j = int64(t)
		case int64:
			j = t
		default:
			return nil, xerrors.Errorf("`height` must be int-like; %T", t)
		}

		h := base.Height(j)
		if err := h.IsValid(nil); err != nil {
			return nil, xerrors.Errorf("invalid height value, %v", v)
		}
		heights[i] = h
	}

	return heights, nil
}
