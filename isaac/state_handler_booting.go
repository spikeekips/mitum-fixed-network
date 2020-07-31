package isaac

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/base/policy"
	"github.com/spikeekips/mitum/base/seal"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/storage"
	"github.com/spikeekips/mitum/util/localtime"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type StateBootingHandler struct {
	*BaseStateHandler
	suffrage    base.Suffrage
	policyTimer *localtime.CallbackTimer
}

func NewStateBootingHandler(
	localstate *Localstate,
	suffrage base.Suffrage,
) (*StateBootingHandler, error) {
	cs := &StateBootingHandler{
		BaseStateHandler: NewBaseStateHandler(localstate, nil, base.StateBooting),
		suffrage:         suffrage,
	}
	cs.BaseStateHandler.Logging = logging.NewLogging(func(c logging.Context) logging.Emitter {
		return c.Str("module", "consensus-state-booting-handler")
	})

	return cs, nil
}

func (cs *StateBootingHandler) Activate(_ *StateChangeContext) error {
	cs.Lock()
	defer cs.Unlock()

	cs.Log().Debug().Msg("activated")

	var ctx *StateChangeContext
	if c, err := cs.initialize(); err != nil {
		cs.Log().Error().Err(err).Msg("failed to initialize at booting")

		return err
	} else if c != nil {
		ctx = c
	}

	cs.activate()

	if ctx != nil {
		go func() {
			if err := cs.ChangeState(ctx.To(), ctx.Voteproof(), ctx.Ballot()); err != nil {
				cs.Log().Error().Err(err).Msg("ChangeState error")
			}
		}()
	}

	return nil
}

func (cs *StateBootingHandler) Deactivate(_ *StateChangeContext) error {
	cs.Lock()
	defer cs.Unlock()

	cs.deactivate()

	if cs.policyTimer != nil {
		if err := cs.policyTimer.Stop(); err != nil {
			return xerrors.Errorf("failed to stop policy timer: %w", err)
		}
	}

	cs.policyTimer = nil

	cs.Log().Debug().Msg("deactivated")

	return nil
}

func (cs *StateBootingHandler) NewSeal(sl seal.Seal) error {
	l := loggerWithSeal(sl, cs.Log())
	l.Debug().Msg("got Seal")

	return nil
}

func (cs *StateBootingHandler) NewVoteproof(voteproof base.Voteproof) error {
	l := loggerWithVoteproofID(voteproof, cs.Log())

	l.Debug().Msg("got Voteproof")

	return nil
}

func (cs *StateBootingHandler) initialize() (*StateChangeContext, error) {
	cs.Log().Debug().Msg("trying to initialize")

	if err := cs.checkBlock(); err != nil {
		cs.Log().Error().Err(err).Msg("something wrong to check blocks")

		if storage.IsNotFoundError(err) {
			if ctx, err0 := cs.whenEmptyBlocks(); err0 != nil {
				return nil, err0
			} else if ctx != nil {
				return ctx, nil
			}

			return nil, nil
		}

		return nil, err
	}

	cs.Log().Debug().Msg("initialized; moves to joining")

	return NewStateChangeContext(base.StateBooting, base.StateJoining, nil, nil), nil
}

func (cs *StateBootingHandler) checkBlock() error {
	cs.Log().Debug().Msg("trying to check block")
	defer cs.Log().Debug().Msg("checked block")

	if blk, found, err := cs.localstate.Storage().LastBlock(); !found {
		return storage.NotFoundError.Errorf("empty block")
	} else if err != nil {
		return err
	} else if err := blk.IsValid(cs.localstate.Policy().NetworkID()); err != nil {
		// TODO if invalid block, it should be re-synced.
		return err
	} else {
		cs.Log().Debug().Hinted("block", blk.Manifest()).Msg("valid initial block found")
	}

	return nil
}

func (cs *StateBootingHandler) whenEmptyBlocks() (*StateChangeContext, error) {
	if len(cs.suffrage.Nodes()) < 2 {
		return nil, xerrors.Errorf("empty block, but no other nodes; can not sync")
	}

	var nodes []network.Node
	for _, a := range cs.suffrage.Nodes() {
		if a.Equal(cs.localstate.Node().Address()) {
			continue
		} else if n, found := cs.localstate.Nodes().Node(a); !found {
			return nil, xerrors.Errorf("unknown node, %s found in suffrage", a)
		} else {
			nodes = append(nodes, n)
		}
	}

	if len(nodes) < 1 {
		return nil, xerrors.Errorf("empty nodes for syncing")
	}

	if ch, err := cs.newPolicyTimer(nodes); err != nil {
		return nil, err
	} else {
		po := <-ch
		if err := cs.localstate.Policy().Merge(po); err != nil {
			return nil, err
		}

		cs.Log().Debug().Interface("policy", po).Msg("got policy at first time and merged")
	}

	return NewStateChangeContext(base.StateBooting, base.StateSyncing, nil, nil), nil
}

// newPolicyTimer starts new timer for gathering NodeInfo from suffrage nodes.
// If nothing to be collected, keeps trying.
func (cs *StateBootingHandler) newPolicyTimer(nodes []network.Node) (
	chan policy.Policy, error) {
	gotPolicyChan := make(chan policy.Policy)

	var once sync.Once
	var called int64
	timer, err := localtime.NewCallbackTimer(
		TimerIDNodeInfo,
		func() (bool, error) {
			cs.Log().Debug().Msg("trying to gather node info")

			var ni policy.Policy
			switch n, err := cs.gatherPolicy(nodes); {
			case err != nil:
				cs.Log().Error().Err(err).Msg("failed to get node info")

				return true, nil
			default:
				cs.Log().Debug().Interface("node_info", n).Msg("got node info")
				ni = n
			}

			once.Do(func() {
				gotPolicyChan <- ni
			})

			return false, nil
		},
		0,
		func() time.Duration {
			if atomic.LoadInt64(&called) < 1 {
				atomic.AddInt64(&called, 1)
				return time.Nanosecond
			}

			return time.Second * 1
		},
	)
	if err != nil {
		return nil, err
	}
	_ = timer.SetLogger(cs.Log())

	if cs.policyTimer != nil {
		if err := cs.policyTimer.Stop(); err != nil {
			return nil, xerrors.Errorf("failed to stop policy timer: %w", err)
		}
	}

	if err := timer.Start(); err != nil {
		return nil, err
	}

	cs.policyTimer = timer

	return gotPolicyChan, nil
}

func (cs *StateBootingHandler) gatherPolicy(nodes []network.Node) (policy.Policy, error) {
	var nis []network.NodeInfo
	for i := range nodes {
		n := nodes[i]
		switch i, err := n.Channel().NodeInfo(); {
		case err != nil:
			cs.Log().Error().Err(err).Hinted("target_node", n.Address()).Msg("failed to get node info from node")

			return nil, err
		case i == nil:
			cs.Log().Error().Err(err).Hinted("target_node", n.Address()).Msg("got empty node info from node")

			continue
		default:
			nis = append(nis, i)
		}
	}

	if len(nis) < 1 {
		return nil, xerrors.Errorf("empty node info")
	}

	set := make([]string, len(nis))
	mnis := map[string]policy.Policy{}

	for i := range nis {
		p := nis[i].Policy()
		if p == nil {
			continue
		}

		h := p.Hash().String()
		set[i] = h
		mnis[h] = p
	}

	var threshold base.Threshold
	if t, err := base.NewThreshold(uint(len(nis)), base.ThresholdRatio(67)); err != nil {
		return nil, err
	} else {
		threshold = t
	}

	if r, key := base.FindMajorityFromSlice(threshold.Total, threshold.Threshold, set); r != base.VoteResultMajority {
		return nil, nil
	} else {
		return mnis[key], nil
	}
}
