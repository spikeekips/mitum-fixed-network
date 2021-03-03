package process

import (
	"context"

	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
)

type HookHandlerProposalProcessorConfig func(context.Context, map[string]interface{}) (config.ProposalProcessor, error)

var DefaultHookHandlersProposalProcessorConfig = map[string]HookHandlerProposalProcessorConfig{
	"default": ProposalProcessorConfigHandlerDefault,
	"error":   ErrorProposalProcessorConfigHandler,
}

func HookProposalProcessorConfigFunc(handlers map[string]HookHandlerProposalProcessorConfig) pm.ProcessFunc {
	return func(ctx context.Context) (context.Context, error) {
		var conf config.LocalNode
		var sc map[string]interface{}
		if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
			return nil, err
		} else {
			sc = conf.Source()
		}

		var m map[string]interface{}
		var st string
		if n, err := config.ParseMap(sc, "proposal-processor", true); err != nil {
			return ctx, err
		} else if t, err := config.ParseType(n, true); err != nil {
			return ctx, err
		} else {
			m = n
			st = t
		}

		var pp config.ProposalProcessor
		if len(st) < 1 {
			if i, err := ProposalProcessorConfigHandlerDefault(ctx, nil); err != nil {
				return ctx, err
			} else {
				pp = i
			}
		} else if h, found := handlers[st]; !found {
			return nil, xerrors.Errorf("unknown proposal-processor found, %s", st)
		} else if i, err := h(ctx, m); err != nil {
			return nil, err
		} else {
			pp = i
		}

		if err := conf.SetProposalProcessor(pp); err != nil {
			return ctx, err
		} else {
			return ctx, nil
		}
	}
}

func ProposalProcessorConfigHandlerDefault(context.Context, map[string]interface{}) (config.ProposalProcessor, error) {
	return config.DefaultProposalProcessor{}, nil
}

func ErrorProposalProcessorConfigHandler(
	_ context.Context,
	m map[string]interface{},
) (config.ProposalProcessor, error) {
	var preparePoints []config.ErrorPoint
	if w, found := m["when-prepare"]; !found {
	} else if p, err := parseErrorPoints(w); err != nil {
		return nil, err
	} else {
		preparePoints = p
	}

	var savePoints []config.ErrorPoint
	if w, found := m["when-save"]; !found {
	} else if p, err := parseErrorPoints(w); err != nil {
		return nil, err
	} else {
		savePoints = p
	}

	return config.ErrorProposalProcessor{
		WhenPreparePoints: preparePoints,
		WhenSavePoints:    savePoints,
	}, nil
}

func parseErrorPoints(v interface{}) ([]config.ErrorPoint, error) {
	var eps []config.ErrorPoint

	if b, err := yaml.Marshal(v); err != nil {
		return nil, err
	} else if err := yaml.Unmarshal(b, &eps); err != nil {
		return nil, xerrors.Errorf("invalid []ErrorPoint: %w", err)
	}

	return eps, nil
}
