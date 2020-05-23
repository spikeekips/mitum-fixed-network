package contestlib

import (
	"sync"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/xerrors"

	mongodbstorage "github.com/spikeekips/mitum/storage/mongodb"
	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
)

type ConditionsChecker struct {
	sync.RWMutex
	*logging.Logging
	client     *mongodbstorage.Client
	collection string
	conditions []*Condition
	remains    []*Condition
	current    int
}

func NewConditionsChecker(
	client *mongodbstorage.Client,
	collection string,
	conditions []*Condition,
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

func (cc *ConditionsChecker) next() (*Condition, bool) {
	cc.RLock()
	defer cc.RUnlock()

	if len(cc.remains) < 1 {
		return nil, false
	}

	return cc.remains[0], true
}

func (cc *ConditionsChecker) Check(exitChan chan error) (bool, error) {
	c, exists := cc.next()
	if !exists {
		cc.Log().Info().Msg("no more conditions; all satisfied")

		return true, nil
	}

	l := cc.Log().WithLogger(func(ctx logging.Context) logging.Emitter {
		return ctx.Str("condition", c.String())
	})

	l.Verbose().Msg("checking condition")

	if passed, err := cc.check(c); err != nil {
		return false, xerrors.Errorf("failed to check: %w", err)
	} else if passed {
		l.Info().Msg("condition matched")

		if c.Action() != nil {
			l.Verbose().Msg("action found")

			go func(action ConditionAction) {
				if err := action.Run(); err != nil {
					l.Error().Err(err).Msg("failed to run action")

					exitChan <- err
				}
			}(c.Action())
		}

		if _, hasNext := cc.next(); !hasNext {
			cc.Log().Info().Msg("all condition are matched")

			return true, nil
		}
	}

	return false, nil
}

func (cc *ConditionsChecker) lastID() (interface{}, error) {
	var lastID interface{}
	if err := cc.client.GetByFilter(
		cc.collection,
		bson.D{},
		func(res *mongo.SingleResult) error {
			var doc struct {
				ID interface{} `bson:"_id"`
			}
			if err := res.Decode(&doc); err != nil {
				return err
			} else {
				lastID = doc.ID
			}

			return nil
		},
		options.FindOne().SetSort(util.NewBSONFilter("_id", -1).D()),
	); err != nil {
		return nil, err
	}

	return lastID, nil
}

func (cc *ConditionsChecker) check(c *Condition) (bool, error) {
	if cc.current == 0 {
		cc.Log().Debug().Str("condition", c.String()).Msg("current condition")
	}

	if lastID, err := cc.lastID(); err != nil {
		return false, err
	} else {
		defer func() {
			c.SetLastID(lastID)
		}()
	}

	var passed bool
	if count, err := cc.client.Count(cc.collection, c.Query(), options.Count().SetLimit(1)); err != nil {
		return false, err
	} else if count > 0 {
		passed = true
	}

	if passed {
		cc.Lock()
		cc.remains = cc.remains[1:]
		cc.current++
		cc.Unlock()

		cc.Log().Debug().Str("condition", c.String()).Msg("current condition")
	}

	return passed, nil
}
