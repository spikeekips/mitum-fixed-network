package config

import (
	"fmt"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

func (no BaseRateLimit) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{
		"preset": no.preset,
	}

	for i := range no.rules {
		r := no.rules[i]
		m[r.Target()] = r
	}

	return jsonenc.Marshal(m)
}

func (no BaseRateLimitRules) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}

	for i := range no.rules {
		r := no.rules[i]
		m[i] = fmt.Sprintf("%d/%s", r.Limit, r.Period.String())
	}

	return jsonenc.Marshal(m)
}

func (no BaseRateLimitTargetRule) MarshalJSON() ([]byte, error) {
	m := map[string]interface{}{}
	if len(no.preset) > 0 {
		m["preset"] = no.preset
	}

	for i := range no.rules {
		r := no.rules[i]
		m[i] = fmt.Sprintf("%d/%s", r.Limit, r.Period.String())
	}

	return jsonenc.Marshal(m)
}
