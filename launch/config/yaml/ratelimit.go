package yamlconfig

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spikeekips/mitum/launch/config"
	"github.com/spikeekips/mitum/util"
	"github.com/ulule/limiter/v3"
	"gopkg.in/yaml.v3"
)

var reNoDigitDuration = regexp.MustCompile(`^(?i)[a-z][a-z]*$`)

type RateLimit struct {
	preset map[string]RateLimitRateRuleSet
	rules  []RateLimitTargetRule
	cache  *string
}

func (no RateLimit) Set(ctx context.Context) (context.Context, error) {
	var l config.LocalNode
	if err := config.LoadConfigContextValue(ctx, &l); err != nil {
		return ctx, err
	}
	conf := l.Network()

	rconf := conf.RateLimit()
	if rconf == nil {
		rconf = config.NewBaseRateLimit(nil)
	}

	i, err := no.setFixed(ctx, rconf)
	if err != nil {
		return ctx, err
	}
	ctx = i

	c, err := no.setRules(ctx, rconf)
	if err != nil {
		return ctx, err
	}
	ctx = c

	if err := conf.SetRateLimit(rconf); err != nil {
		return ctx, err
	}

	return ctx, nil
}

func (no RateLimit) setFixed(ctx context.Context, conf config.RateLimit) (context.Context, error) {
	preset := map[string]config.RateLimitRules{}
	for i := range no.preset {
		p := no.preset[i]
		rs := map[string]limiter.Rate{}
		for j := range p {
			rs[j] = p[j].rate()
		}

		preset[i] = config.NewBaseRateLimitRules(rs)
	}

	if err := conf.SetPreset(preset); err != nil {
		return ctx, err
	}

	if no.cache != nil {
		if err := conf.SetCache(*no.cache); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

func (no RateLimit) setRules(ctx context.Context, conf config.RateLimit) (context.Context, error) {
	if len(no.rules) < 1 {
		return ctx, nil
	}

	rules := make([]config.RateLimitTargetRule, len(no.rules))
	for i := range no.rules {
		r := no.rules[i]
		tr := config.NewBaseRateLimitTargetRule(r.target, r.preset)

		rs := map[string]limiter.Rate{}
		for j := range r.rules {
			rs[j] = r.rules[j].rate()
		}

		if err := tr.SetRules(rs); err != nil {
			return ctx, err
		}
		rules[i] = tr
	}

	if err := conf.SetRules(rules); err != nil {
		return ctx, err
	}

	return ctx, nil
}

func (no RateLimit) MarshalYAML() (interface{}, error) {
	m := map[string]interface{}{
		"preset": no.preset,
	}

	if no.cache != nil {
		m["cache"] = *no.cache
	}

	for i := range no.rules {
		j := no.rules[i]

		m[j.target] = j
	}

	return m, nil
}

func (no *RateLimit) UnmarshalYAML(value *yaml.Node) error {
	var fixed struct {
		Preset map[string]RateLimitRateRuleSet `yaml:"preset,omitempty"`
		Cache  *string                         `yaml:"cache,omitempty"`
	}

	if err := value.Decode(&fixed); err != nil {
		return err
	}
	no.preset = fixed.Preset
	no.cache = fixed.Cache

	// NOTE RateLimit keeps the defined order of set
	i, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	r := bytes.NewBuffer(i)

	var nb []byte
	var skip bool
	if err := util.Readlines(r, func(b []byte) error {
		switch {
		case bytes.HasPrefix(b, []byte("cache:")), bytes.HasPrefix(b, []byte("preset:")):
			skip = true

			return nil
		case skip:
			if bytes.HasPrefix(b, []byte(" ")) { // NOTE tab indentation is forbidden in YAML
				return nil
			}

			skip = false
		}

		if !bytes.HasPrefix(b, []byte(" ")) {
			nb = append(nb, []byte("- ")...)
		}
		nb = append(nb, b...)

		return nil
	}); err != nil {
		return err
	}

	var items []RateLimitTargetRule
	if err := yaml.Unmarshal(nb, &items); err != nil {
		return err
	}

	no.rules = items

	return nil
}

type RateLimitTargetRule struct {
	target string
	preset string
	rules  RateLimitRateRuleSet
}

func (no *RateLimitTargetRule) UnmarshalYAML(value *yaml.Node) error {
	var m map[string]struct {
		Preset *string              `yaml:",omitempty"`
		Rules  RateLimitRateRuleSet `yaml:",inline,omitempty"`
	}

	if err := value.Decode(&m); err != nil {
		return err
	}

	if len(m) < 1 {
		return errors.Errorf("empty set")
	}

	for i := range m {
		r := m[i]

		no.target = i
		if r.Preset != nil {
			no.preset = *r.Preset
		}

		no.rules = r.Rules

		break
	}

	return nil
}

type RateLimitRateRuleSet map[string]RateLimitRate

type RateLimitRate limiter.Rate

func (no RateLimitRate) rate() limiter.Rate {
	return limiter.Rate(no)
}

func (no RateLimitRate) MarshalYAML() (interface{}, error) {
	return fmt.Sprintf("%d/%s", no.Limit, no.Period.String()), nil
}

func (no *RateLimitRate) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	s = strings.ReplaceAll(s, " ", "")
	r := limiter.Rate{Formatted: s}

	var l, p string
	values := strings.Split(s, "/")
	if len(values) != 2 {
		return errors.Errorf("incorrect format '%s'", s)
	}
	l, p = values[0], strings.ToLower(values[1])

	if reNoDigitDuration.MatchString(p) {
		p = "1" + p
	}

	i, err := time.ParseDuration(p)
	if err != nil {
		return err
	}
	r.Period = i

	n, err := strconv.ParseInt(l, 10, 64)
	if err != nil {
		return errors.Errorf("incorrect limit '%s'", l)
	}
	r.Limit = n

	*no = RateLimitRate(r)

	return nil
}
