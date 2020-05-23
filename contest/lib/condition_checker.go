package contestlib

import (
	"sync"

	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util/logging"
)

type ConditionsChecker struct {
	*logging.Logging
	sync.RWMutex
	client     *mongodbstorage.Client
	collection string
	conditions []Condition
	remains    []Condition
}

func NewConditionsChecker(
	client *mongodbstorage.Client,
	collection string,
	conditions []Condition,
) *ConditionsChecker {
	return &ConditionsChecker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "condition-checker")
		}),
		client:     client,
		collection: collection,
		conditions: conditions,
		remains:    conditions,
	}
}

func (cc *ConditionsChecker) next() (Condition, bool) {
	cc.RLock()
	defer cc.RUnlock()

	if len(cc.remains) < 1 {
		return Condition{}, false
	}

	return cc.remains[0], true
}

func (cc *ConditionsChecker) Check() (bool, error) {
	c, exists := cc.next()
	if !exists {
		cc.Log().Debug().Msg("no more conditions; all satisfied")

		return true, nil
	}

	l := cc.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("condition", c.String())
	})

	l.Debug().Msg("checking condition")

	if passed, err := cc.check(c); err != nil {
		return false, xerrors.Errorf("failed to check: %w", err)
	} else if passed {
		l.Debug().Msg("condition matched")

		if _, hasNext := cc.next(); !hasNext {
			cc.Log().Debug().Msg("all condition are matched")

			return true, nil
		}
	}

	return false, nil
}

func (cc *ConditionsChecker) check(c Condition) (bool, error) {
	// TODO should remember the last checked object id
	var passed bool
	if count, err := cc.client.Count(cc.collection, c.Query()); err != nil {
		return false, err
	} else if count > 0 {
		passed = true
	}

	if passed {
		cc.Lock()
		cc.remains = cc.remains[1:]
		cc.Unlock()
	}

	return passed, nil
}
