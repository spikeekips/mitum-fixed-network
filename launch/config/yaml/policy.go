package yamlconfig

import (
	"context"

	"github.com/spikeekips/mitum/launch/config"
)

type Policy struct {
	ThresholdRatio                   *float64               `yaml:"threshold,omitempty"`
	TimeoutWaitingProposal           *string                `yaml:"timeout-waiting-proposal,omitempty"`
	IntervalBroadcastingINITBallot   *string                `yaml:"interval-broadcasting-init-ballot,omitempty"`
	IntervalBroadcastingProposal     *string                `yaml:"interval-broadcasting-proposal,omitempty"`
	WaitBroadcastingACCEPTBallot     *string                `yaml:"wait-broadcasting-accept-ballot,omitempty"`
	IntervalBroadcastingACCEPTBallot *string                `yaml:"interval-broadcasting-accept-ballot,omitempty"`
	TimespanValidBallot              *string                `yaml:"timespan-valid-ballot,omitempty"`
	TimeoutProcessProposal           *string                `yaml:"timeout-process-proposal,omitempty"`
	Extras                           map[string]interface{} `yaml:",inline"`
}

func (no Policy) Set(ctx context.Context) (context.Context, error) {
	var l config.LocalNode
	var conf config.Policy
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	} else {
		conf = l.Policy()
	}

	if no.ThresholdRatio != nil {
		if err := conf.SetThresholdRatio(*no.ThresholdRatio); err != nil {
			return ctx, err
		}
	}
	if no.TimeoutWaitingProposal != nil {
		if err := conf.SetTimeoutWaitingProposal(*no.TimeoutWaitingProposal); err != nil {
			return ctx, err
		}
	}
	if no.IntervalBroadcastingINITBallot != nil {
		if err := conf.SetIntervalBroadcastingINITBallot(*no.IntervalBroadcastingINITBallot); err != nil {
			return ctx, err
		}
	}
	if no.IntervalBroadcastingProposal != nil {
		if err := conf.SetIntervalBroadcastingProposal(*no.IntervalBroadcastingProposal); err != nil {
			return ctx, err
		}
	}
	if no.WaitBroadcastingACCEPTBallot != nil {
		if err := conf.SetWaitBroadcastingACCEPTBallot(*no.WaitBroadcastingACCEPTBallot); err != nil {
			return ctx, err
		}
	}
	if no.IntervalBroadcastingACCEPTBallot != nil {
		if err := conf.SetIntervalBroadcastingACCEPTBallot(*no.IntervalBroadcastingACCEPTBallot); err != nil {
			return ctx, err
		}
	}
	if no.TimespanValidBallot != nil {
		if err := conf.SetTimespanValidBallot(*no.TimespanValidBallot); err != nil {
			return ctx, err
		}
	}
	if no.TimeoutProcessProposal != nil {
		if err := conf.SetTimeoutProcessProposal(*no.TimeoutProcessProposal); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}
