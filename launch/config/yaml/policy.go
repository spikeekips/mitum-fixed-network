package yamlconfig

import (
	"context"
	"reflect"

	"github.com/spikeekips/mitum/launch/config"
)

type Policy struct {
	ThresholdRatio                   *float64               `yaml:"threshold,omitempty"`
	MaxOperationsInSeal              *uint                  `yaml:"max-operations-in-seal"`
	MaxOperationsInProposal          *uint                  `yaml:"max-operations-in-proposal"`
	TimeoutWaitingProposal           *string                `yaml:"timeout-waiting-proposal,omitempty"`
	IntervalBroadcastingINITBallot   *string                `yaml:"interval-broadcasting-init-ballot,omitempty"`
	IntervalBroadcastingProposal     *string                `yaml:"interval-broadcasting-proposal,omitempty"`
	WaitBroadcastingACCEPTBallot     *string                `yaml:"wait-broadcasting-accept-ballot,omitempty"`
	IntervalBroadcastingACCEPTBallot *string                `yaml:"interval-broadcasting-accept-ballot,omitempty"`
	TimespanValidBallot              *string                `yaml:"timespan-valid-ballot,omitempty"`
	NetworkConnectionTimeout         *string                `yaml:"network-connection-timeout,omitempty"`
	NetworkConnectionTLSInsecure     *bool                  `yaml:"network-connection-tls-insecure,omitempty"`
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

	if err := no.setUints(conf); err != nil {
		return ctx, err
	}

	if err := no.setDurations(conf); err != nil {
		return ctx, err
	}

	if no.NetworkConnectionTLSInsecure != nil {
		if err := conf.SetNetworkConnectionTLSInsecure(*no.NetworkConnectionTLSInsecure); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

func (no Policy) setUints(conf config.Policy) error {
	uintCol := [][2]interface{}{
		{no.MaxOperationsInSeal, conf.SetMaxOperationsInSeal},
		{no.MaxOperationsInProposal, conf.SetMaxOperationsInProposal},
	}

	for i := range uintCol {
		v, f := uintCol[i][0], uintCol[i][1]

		rv := reflect.ValueOf(v)
		if rv.IsNil() {
			continue
		}

		if err := f.(func(uint) error)(rv.Elem().Interface().(uint)); err != nil {
			return err
		}
	}

	return nil
}

func (no Policy) setDurations(conf config.Policy) error {
	durationCol := [][2]interface{}{
		{no.TimeoutWaitingProposal, conf.SetTimeoutWaitingProposal},
		{no.IntervalBroadcastingINITBallot, conf.SetIntervalBroadcastingINITBallot},
		{no.IntervalBroadcastingProposal, conf.SetIntervalBroadcastingProposal},
		{no.WaitBroadcastingACCEPTBallot, conf.SetWaitBroadcastingACCEPTBallot},
		{no.IntervalBroadcastingACCEPTBallot, conf.SetIntervalBroadcastingACCEPTBallot},
		{no.TimespanValidBallot, conf.SetTimespanValidBallot},
		{no.NetworkConnectionTimeout, conf.SetNetworkConnectionTimeout},
	}

	for i := range durationCol {
		v, f := durationCol[i][0], durationCol[i][1]

		rv := reflect.ValueOf(v)
		if rv.IsNil() {
			continue
		}

		if err := f.(func(string) error)(rv.Elem().Interface().(string)); err != nil {
			return err
		}
	}

	return nil
}
