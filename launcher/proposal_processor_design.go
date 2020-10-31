package launcher

import (
	"fmt"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
)

type BlockPoint struct {
	Height base.Height
	Round  base.Round
}

func ParseBlockPoint(s string) (BlockPoint, error) {
	var h int64
	var r uint64
	if n, err := fmt.Sscanf(s, "%d,%d", &h, &r); err != nil {
		return BlockPoint{}, xerrors.Errorf("invalid block point string: %v: %w", s, err)
	} else if n != 2 {
		return BlockPoint{}, xerrors.Errorf("invalid block point string: %v", s)
	}

	height := base.Height(h)
	if err := height.IsValid(nil); err != nil {
		return BlockPoint{}, err
	}

	return BlockPoint{
		Height: height,
		Round:  base.Round(r),
	}, nil
}

func ParseBlockPoints(hs interface{}) ([]BlockPoint, error) {
	l, ok := hs.([]interface{})
	if !ok {
		return nil, xerrors.Errorf("blockpoints must be list; %T", hs)
	}
	bps := make([]BlockPoint, len(l))

	for i, v := range l {
		if s, ok := v.(string); !ok {
			return nil, xerrors.Errorf("invalid BlockPoint string, %v", v)
		} else if bp, err := ParseBlockPoint(s); err != nil {
			return nil, err
		} else {
			bps[i] = bp
		}
	}

	return bps, nil
}

type ProposalProcessorDesign struct {
	Type              string
	Info              map[string]interface{} `yaml:"-"`
	errorINITPoints   []BlockPoint
	errorACCEPTPoints []BlockPoint
	wrongPoints       []BlockPoint
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
	case "wrong-processing":
		var points []BlockPoint
		if i, found := st.Info["points"]; !found {
		} else if hs, err := ParseBlockPoints(i); err != nil {
			return xerrors.Errorf("invalid points for wrong points: %w", err)
		} else {
			points = hs
		}

		if len(points) < 1 {
			return xerrors.Errorf("accept or init points must be set for wrong-processing")
		}

		st.wrongPoints = points
	case "occurring-error":
		var initPoints, acceptPoints []BlockPoint
		if i, found := st.Info["init-points"]; !found {
		} else if hs, err := ParseBlockPoints(i); err != nil {
			return xerrors.Errorf("invalid points for init error points: %w", err)
		} else {
			initPoints = hs
		}
		if i, found := st.Info["accept-points"]; !found {
		} else if hs, err := ParseBlockPoints(i); err != nil {
			return xerrors.Errorf("invalid points for accept error points: %w", err)
		} else {
			acceptPoints = hs
		}

		if len(initPoints) < 1 && len(acceptPoints) < 1 {
			return xerrors.Errorf("accept or init points must be set for occurring-error")
		}

		st.errorINITPoints = initPoints
		st.errorACCEPTPoints = acceptPoints
	default:
		return xerrors.Errorf("unknown type, %q", st.Type)
	}

	return nil
}

func (st *ProposalProcessorDesign) New(
	local *isaac.Local, suffrage base.Suffrage,
) (isaac.ProposalProcessor, error) {
	switch st.Type {
	case "default":
		return isaac.NewDefaultProposalProcessor(local, suffrage), nil
	case "wrong-processing":
		return NewWrongProposalProcessor(local, suffrage, st.wrongPoints), nil
	case "occurring-error":
		return NewErrorProposalProcessor(local, suffrage, st.errorINITPoints, st.errorACCEPTPoints), nil
	default:
		return nil, xerrors.Errorf("unknown type found: %v", st.Type)
	}
}
