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

var EvenCollection = "event"

type ConditionsChecker struct {
	sync.RWMutex
	*logging.Logging
	defaultClient *mongodbstorage.Client
	clients       map[string]*mongodbstorage.Client
	conditions    []*Condition
	remains       []*Condition
	current       int
	vars          *Vars
}

func NewConditionsChecker(
	client *mongodbstorage.Client,
	conditions []*Condition,
	vars *Vars,
) *ConditionsChecker {
	return &ConditionsChecker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "condition-checker")
		}),
		defaultClient: client,
		clients: map[string]*mongodbstorage.Client{
			"": client,
		},
		conditions: conditions,
		remains:    conditions,
		vars:       vars,
	}
}

func (cc *ConditionsChecker) next() (*Condition, bool) {
	cc.RLock()
	defer cc.RUnlock()

	if len(cc.remains) < 1 {
		return nil, false
	}

	c := cc.remains[0]
	if _, err := c.FormatQuery(cc.vars); err != nil {
		return nil, false
	}

	return c, true
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
			l.Debug().Msg("action found")

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

func (cc *ConditionsChecker) check(c *Condition) (bool, error) {
	var record map[string]interface{}
	switch r, err := cc.query(c); {
	case err != nil:
		return false, err
	case r == nil:
		return false, nil
	default:
		record = r
		cc.vars.Set("last_matched", record["_id"])
	}

	if len(c.Register) > 0 {
		for _, r := range c.Register {
			if v, found := Lookup(record, r.Key); !found {
				continue
			} else {
				cc.vars.Set(r.Assign, v)
			}
		}
	}

	cc.Lock()
	cc.remains = cc.remains[1:]
	cc.current++
	cc.Unlock()

	if n, found := cc.next(); found {
		cc.Log().Debug().Str("condition", n.String()).Msg("current condition")
	} else {
		cc.Log().Debug().Msg("no more condition")
	}

	return true, nil
}

func (cc *ConditionsChecker) client(s string) *mongodbstorage.Client {
	f := func(s string) *mongodbstorage.Client {
		cc.RLock()
		defer cc.RUnlock()

		if c, found := cc.clients[s]; found {
			return c
		}

		return nil
	}

	if len(s) < 1 {
		return cc.defaultClient
	} else if c := f(s); c != nil {
		return c
	}

	cc.Lock()
	defer cc.Unlock()

	c := cc.defaultClient.New(s)
	cc.clients[s] = c

	return c
}

func (cc *ConditionsChecker) query(c *Condition) (map[string]interface{}, error) {
	var query bson.M
	if q, err := c.FormatQuery(cc.vars); err != nil {
		return nil, err
	} else {
		query = q
	}

	client := cc.client(c.DB())

	var record map[string]interface{}
	if err := client.Find(c.Collection(), query, func(cursor *mongo.Cursor) (bool, error) {
		if err := cursor.Decode(&record); err != nil {
			return false, err
		}

		return false, nil
	},
		options.Find().SetSort(util.NewBSONFilter("_id", -1).D()).SetLimit(1),
	); err != nil {
		return nil, err
	}

	return record, nil
}
