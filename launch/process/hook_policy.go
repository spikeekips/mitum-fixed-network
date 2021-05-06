package process

import (
	"context"

	"github.com/spikeekips/mitum/base"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/launch/config"
)

const HookNameSetPolicy = "set_policy"

func HookSetPolicy(ctx context.Context) (context.Context, error) {
	var networkID base.NetworkID
	var l config.LocalNode
	var conf config.Policy
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	} else {
		networkID = l.NetworkID()
		conf = l.Policy()
	}

	policy := isaac.NewLocalPolicy(networkID)

	_ = policy.SetThresholdRatio(conf.ThresholdRatio())
	if _, err := policy.SetMaxOperationsInSeal(conf.MaxOperationsInSeal()); err != nil {
		return ctx, err
	}
	if _, err := policy.SetMaxOperationsInProposal(conf.MaxOperationsInProposal()); err != nil {
		return ctx, err
	}
	if _, err := policy.SetTimeoutWaitingProposal(conf.TimeoutWaitingProposal()); err != nil {
		return ctx, err
	}
	if _, err := policy.SetIntervalBroadcastingINITBallot(conf.IntervalBroadcastingINITBallot()); err != nil {
		return ctx, err
	}
	if _, err := policy.SetIntervalBroadcastingProposal(conf.IntervalBroadcastingProposal()); err != nil {
		return ctx, err
	}
	if _, err := policy.SetWaitBroadcastingACCEPTBallot(conf.WaitBroadcastingACCEPTBallot()); err != nil {
		return ctx, err
	}
	if _, err := policy.SetIntervalBroadcastingACCEPTBallot(conf.IntervalBroadcastingACCEPTBallot()); err != nil {
		return ctx, err
	}
	if _, err := policy.SetTimespanValidBallot(conf.TimespanValidBallot()); err != nil {
		return ctx, err
	}
	if _, err := policy.SetNetworkConnectionTimeout(conf.NetworkConnectionTimeout()); err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, ContextValuePolicy, policy), nil
}
