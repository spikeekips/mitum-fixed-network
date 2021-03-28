package config

import (
	"context"
	"time"

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

	if conf.Cache() == nil {
		if err := conf.SetCache(DefaultLocalNetworkCache); err != nil {
			return false, err
		}
	}

	if conf.SealCache() == nil {
		if err := conf.SetSealCache(DefaultLocalNetworkSealCache); err != nil {
			return false, err
		}
	}

	return true, nil
}

func (cc *checker) CheckStorage() (bool, error) {
	conf := cc.config.Storage()

	if len(conf.BlockData().Path()) < 1 {
		if err := conf.BlockData().SetPath(DefaultBlockDataPath); err != nil {
			return false, err
		}
	}

	if conf.Database().URI() == nil {
		if err := conf.Database().SetURI(DefaultDatabaseURI); err != nil {
			return false, err
		}
	}
	if conf.Database().Cache() == nil {
		if err := conf.Database().SetCache(DefaultDatabaseCache); err != nil {
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

	uints := [][3]interface{}{
		{conf.MaxOperationsInSeal(), conf.SetMaxOperationsInSeal, isaac.DefaultPolicyMaxOperationsInSeal},             // nolint:lll
		{conf.MaxOperationsInProposal(), conf.SetMaxOperationsInProposal, isaac.DefaultPolicyMaxOperationsInProposal}, // nolint:lll
	}

	for i := range uints {
		v := uints[i][0].(uint)
		d := uints[i][2].(uint)
		f := uints[i][1].(func(uint) error)

		if v > 0 {
			continue
		}
		if err := f(d); err != nil {
			return false, err
		}
	}

	durs := [][3]interface{}{
		{conf.TimeoutWaitingProposal(), conf.SetTimeoutWaitingProposal, isaac.DefaultPolicyTimeoutWaitingProposal},                               // nolint:lll
		{conf.IntervalBroadcastingINITBallot(), conf.SetIntervalBroadcastingINITBallot, isaac.DefaultPolicyIntervalBroadcastingINITBallot},       // nolint:lll
		{conf.IntervalBroadcastingProposal(), conf.SetIntervalBroadcastingProposal, isaac.DefaultPolicyIntervalBroadcastingProposal},             // nolint:lll
		{conf.WaitBroadcastingACCEPTBallot(), conf.SetWaitBroadcastingACCEPTBallot, isaac.DefaultPolicyWaitBroadcastingACCEPTBallot},             // nolint:lll
		{conf.IntervalBroadcastingACCEPTBallot(), conf.SetIntervalBroadcastingACCEPTBallot, isaac.DefaultPolicyIntervalBroadcastingACCEPTBallot}, // nolint:lll
		{conf.TimespanValidBallot(), conf.SetTimespanValidBallot, isaac.DefaultPolicyTimespanValidBallot},                                        // nolint:lll
		{conf.TimeoutProcessProposal(), conf.SetTimeoutProcessProposal, isaac.DefaultPolicyTimeoutProcessProposal},                               // nolint:lll
		{conf.NetworkConnectionTimeout(), conf.SetNetworkConnectionTimeout, isaac.DefaultPolicyNetworkConnectionTimeout},                         // nolint:lll
	}

	for i := range durs {
		v := durs[i][0].(time.Duration)
		d := durs[i][2].(time.Duration)
		f := durs[i][1].(func(string) error)

		if v > 0 {
			continue
		}
		if err := f(d.String()); err != nil {
			return false, err
		}
	}

	if err := conf.SetNetworkConnectionTLSInsecure(conf.NetworkConnectionTLSInsecure()); err != nil {
		return false, err
	}

	return true, nil
}
