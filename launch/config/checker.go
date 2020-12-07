package config

import (
	"context"

	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/util/logging"
)

type checker struct {
	*logging.Logging
	ctx    context.Context
	config LocalNode
}

func NewChecker(ctx context.Context) (*checker, error) {
	var conf LocalNode
	if err := LoadConfigContextValue(ctx, &conf); err != nil {
		return nil, err
	}

	cc := &checker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "config-checker")
		}),
		ctx:    ctx,
		config: conf,
	}

	var l logging.Logger
	if err := LoadLogContextValue(ctx, &l); err == nil {
		_ = cc.SetLogger(l)
	}

	return cc, nil
}

func (cc *checker) Context() context.Context {
	return cc.ctx
}

func (cc *checker) CheckLocalNetwork() (bool, error) {
	conf := cc.config.Network()
	if conf.URL() == nil {
		if err := conf.SetURL(DefaultLocalNetworkURL.String()); err != nil {
			return false, err
		}
	}

	if conf.Bind() == nil {
		if err := conf.SetBind(DefaultLocalNetworkBind.String()); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (cc *checker) CheckStorage() (bool, error) {
	conf := cc.config.Storage()

	if len(conf.BlockFS().Path()) < 1 {
		if err := conf.BlockFS().SetPath(DefaultBlockFSPath); err != nil {
			return false, err
		}
	}

	if conf.Main().URI() == nil {
		if err := conf.Main().SetURI(DefaultMainStorageURI); err != nil {
			return false, err
		}
	}
	if conf.Main().Cache() == nil {
		if err := conf.Main().SetCache(DefaultMainStorageCache); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (cc *checker) CheckPolicy() (bool, error) {
	conf := cc.config.Policy()

	if conf.ThresholdRatio() == 0 {
		if err := conf.SetThresholdRatio(isaac.DefaultPolicyThresholdRatio.Float64()); err != nil {
			return false, err
		}
	}

	if conf.TimeoutWaitingProposal() == 0 {
		if err := conf.SetTimeoutWaitingProposal(isaac.DefaultPolicyTimeoutWaitingProposal.String()); err != nil {
			return false, err
		}
	}

	if conf.IntervalBroadcastingINITBallot() == 0 {
		if err := conf.SetIntervalBroadcastingINITBallot(
			isaac.DefaultPolicyIntervalBroadcastingINITBallot.String()); err != nil {
			return false, err
		}
	}

	if conf.IntervalBroadcastingProposal() == 0 {
		if err := conf.SetIntervalBroadcastingProposal(isaac.DefaultPolicyIntervalBroadcastingProposal.String()); err != nil {
			return false, err
		}
	}

	if conf.WaitBroadcastingACCEPTBallot() == 0 {
		if err := conf.SetWaitBroadcastingACCEPTBallot(isaac.DefaultPolicyWaitBroadcastingACCEPTBallot.String()); err != nil {
			return false, err
		}
	}

	if conf.IntervalBroadcastingACCEPTBallot() == 0 {
		if err := conf.SetIntervalBroadcastingACCEPTBallot(
			isaac.DefaultPolicyIntervalBroadcastingACCEPTBallot.String()); err != nil {
			return false, err
		}
	}

	if conf.TimespanValidBallot() == 0 {
		if err := conf.SetTimespanValidBallot(isaac.DefaultPolicyTimespanValidBallot.String()); err != nil {
			return false, err
		}
	}

	if conf.TimeoutProcessProposal() == 0 {
		if err := conf.SetTimeoutProcessProposal(isaac.DefaultPolicyTimeoutProcessProposal.String()); err != nil {
			return false, err
		}
	}

	return true, nil
}
