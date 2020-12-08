package process

import (
	"context"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/base/operation"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/launch/pm"
)

type HookHandlerGenesisOperations func(context.Context, map[string]interface{}) (operation.Operation, error)

var (
	DefaultGenesisOperationToken         = []byte("genesis-operation-token")
	DefaultHookHandlersGenesisOperations = map[string]HookHandlerGenesisOperations{}
)

func HookGenesisOperationFunc(handlers map[string]HookHandlerGenesisOperations) pm.ProcessFunc {
	return func(ctx context.Context) (context.Context, error) {
		var conf config.LocalNode
		var sc map[string]interface{}
		if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
			return nil, err
		} else {
			sc = conf.Source()
		}

		var l []map[string]interface{}
		if i, err := parseGenesisOperations(sc["genesis-operations"]); err != nil {
			return ctx, err
		} else {
			l = i
		}

		ops := make([]operation.Operation, len(l))
		for i := range l {
			if t, err := config.ParseType(l[i], false); err != nil {
				return ctx, err
			} else if h, found := handlers[t]; !found {
				return ctx, xerrors.Errorf("invalid genesis operation found,  %q", t)
			} else if op, err := h(ctx, l[i]); err != nil {
				return nil, err
			} else {
				ops[i] = op
			}
		}

		if err := conf.SetGenesisOperations(ops); err != nil {
			return ctx, err
		} else {
			return ctx, nil
		}
	}
}

func parseGenesisOperations(o interface{}) ([]map[string]interface{}, error) {
	if o == nil {
		return nil, nil
	}

	switch l, ok := o.([]interface{}); {
	case !ok:
		return nil, xerrors.Errorf("invalid genesis-operations configs, %T found", o)
	case len(l) < 1:
		return nil, nil
	default:
		ml := make([]map[string]interface{}, len(l))
		for i := range l {
			if m, ok := l[i].(map[string]interface{}); !ok {
				return nil, xerrors.Errorf("invalid genesis operation config type, %T", l[i])
			} else if _, err := config.ParseType(m, false); err != nil {
				return nil, xerrors.Errorf("invalid genesis operation found: %w", err)
			} else {
				ml[i] = m
			}
		}

		return ml, nil
	}
}
