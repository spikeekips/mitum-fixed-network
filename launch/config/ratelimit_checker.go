package config

import (
	"context"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/logging"
	"github.com/ulule/limiter/v3"
	"golang.org/x/xerrors"
)

type RateLimitChecker struct {
	*logging.Logging
	ctx        context.Context
	conf       RateLimit
	basePreset map[string]RateLimitRules
}

func NewRateLimitChecker(
	ctx context.Context,
	conf RateLimit,
	basePreset map[string]RateLimitRules,
) *RateLimitChecker {
	if basePreset == nil {
		basePreset = map[string]RateLimitRules{}
	}

	cc := &RateLimitChecker{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", "config-ratelimit-checker")
		}),
		ctx:        ctx,
		conf:       conf,
		basePreset: basePreset,
	}

	var l logging.Logger
	if err := LoadLogContextValue(ctx, &l); err == nil {
		_ = cc.SetLogger(l)
	}

	return cc
}

func (cc *RateLimitChecker) Context() context.Context {
	return cc.ctx
}

func (cc *RateLimitChecker) Config() RateLimit {
	return cc.conf
}

func (cc *RateLimitChecker) Initialize() (bool, error) {
	if cc.conf == nil {
		return false, nil
	}

	return true, nil
}

func (cc *RateLimitChecker) Check() (bool, error) {
	if err := cc.checkRateLimitPresets(); err != nil {
		return false, err
	}

	if err := cc.checkRateLimitTargetRules(); err != nil {
		return false, err
	}

	return true, nil
}

func (cc *RateLimitChecker) checkRateLimitPresets() error {
	preset := cc.conf.Preset()
	if preset == nil {
		preset = map[string]RateLimitRules{}
	} else if len(preset) > 0 {
		for i := range preset {
			preset[i] = cc.fillRateLimitPreset(i, preset[i])
		}
	}

	for i := range cc.basePreset {
		if _, found := preset[i]; !found {
			preset[i] = cc.basePreset[i]
		}
	}

	return cc.conf.SetPreset(preset)
}

func (cc *RateLimitChecker) fillRateLimitPreset(name string, r RateLimitRules) RateLimitRules {
	var defined map[string]limiter.Rate
	if i, found := cc.basePreset[name]; found {
		defined = i.Rules()
	} else {
		defined = map[string]limiter.Rate{}
	}

	rules := r.Rules()
	for i := range defined {
		if _, found := rules[i]; !found {
			rules[i] = defined[i]
		}
	}

	return NewBaseRateLimitRules(rules)
}

func (cc *RateLimitChecker) checkRateLimitTargetRules() error {
	rules := cc.conf.Rules()
	if len(rules) < 1 {
		return util.IgnoreError.Errorf("empty rules")
	}

	nr := make([]RateLimitTargetRule, len(rules))
	for i := range rules {
		j, err := cc.checkRateLimitTargetRule(rules[i])
		if err != nil {
			return err
		}
		nr[i] = j
	}

	return cc.conf.SetRules(nr)
}

func (cc *RateLimitChecker) checkRateLimitTargetRule(r RateLimitTargetRule) (RateLimitTargetRule, error) {
	if i := r.Target(); len(i) < 1 {
		return nil, xerrors.Errorf("empty target")
	} else if err := r.SetIPNet(i); err != nil {
		return nil, err
	}

	if len(r.Preset()) < 1 {
		return r, nil
	}

	var preset map[string]limiter.Rate
	presets := cc.conf.Preset()
	i, found := presets[r.Preset()]
	if !found {
		return nil, xerrors.Errorf("unknown preset, %q", r.Preset())
	}
	preset = i.Rules()

	rs := map[string]limiter.Rate{}

	rules := r.Rules()
	for i := range preset {
		if j, found := rules[i]; !found {
			rs[i] = preset[i]
		} else {
			rs[i] = j
		}
	}

	return r, r.SetRules(rs)
}
