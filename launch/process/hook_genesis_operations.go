package process

import (
	"context"

	"github.com/pkg/errors"
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
		if err := config.LoadConfigContextValue(ctx, &conf); err != nil {
			return nil, err
		}
		sc := conf.Source()

		l, err := parseGenesisOperations(sc["genesis-operations"])
		if err != nil {
			return ctx, err
		}

		ops := make([]operation.Operation, len(l))
		for i := range l {
			t, err := config.ParseType(l[i], false)
			if err != nil {
				return ctx, err
			}

			h, found := handlers[t]
			switch {
			case !found:
				return ctx, errors.Errorf("invalid genesis operation found,  %q", t)
			case h == nil:
				return ctx, nil
			}

			op, err := h(ctx, l[i])
			if err != nil {
				return nil, err
			}

			ops[i] = op
		}

		if err := conf.SetGenesisOperations(ops); err != nil {
			return ctx, err
		}
		return ctx, nil
	}
}

func parseGenesisOperations(o interface{}) ([]map[string]interface{}, error) {
	if o == nil {
		return nil, nil
	}

	switch l, ok := o.([]interface{}); {
	case !ok:
		return nil, errors.Errorf("invalid genesis-operations configs, %T found", o)
	case len(l) < 1:
		return nil, nil
	default:
		ml := make([]map[string]interface{}, len(l))
		for i := range l {
			if m, ok := l[i].(map[string]interface{}); !ok {
				return nil, errors.Errorf("invalid genesis operation config type, %T", l[i])
			} else if _, err := config.ParseType(m, false); err != nil {
				return nil, errors.Wrap(err, "invalid genesis operation found")
			} else {
				ml[i] = m
			}
		}

		return ml, nil
	}
}
