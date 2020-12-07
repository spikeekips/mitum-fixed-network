package process

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
)

type HookHandlerProposalProcessor func(context.Context, map[string]interface{}) (config.ProposalProcessor, error)

var DefaultHookHandlersProposalProcessor = map[string]HookHandlerProposalProcessor{
	"default": ProposalProcessorHandlerDefault,
}

func HookProposalProcessorFunc(handlers map[string]HookHandlerProposalProcessor) pm.ProcessFunc {
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
			if i, err := ProposalProcessorHandlerDefault(ctx, nil); err != nil {
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

func ProposalProcessorHandlerDefault(context.Context, map[string]interface{}) (config.ProposalProcessor, error) {
	return config.DefaultProposalProcessor{}, nil
}
