package config

import "fmt"

func (no BaseRateLimit) MarshalYAML() (interface{}, error) {
	m := map[string]interface{}{
		"preset": no.preset,
	}

	for i := range no.rules {
		r := no.rules[i]
		m[r.Target()] = r
	}

	return m, nil
}

func (no BaseRateLimitRules) MarshalYAML() (interface{}, error) {
	m := map[string]interface{}{}

	for i := range no.rules {
		r := no.rules[i]
		m[i] = fmt.Sprintf("%d/%s", r.Limit, r.Period.String())
	}

	return m, nil
}

func (no BaseRateLimitTargetRule) MarshalYAML() (interface{}, error) {
	m := map[string]interface{}{}
	if len(no.preset) > 0 {
		m["preset"] = no.preset
	}

	for i := range no.rules {
		r := no.rules[i]
		m[i] = fmt.Sprintf("%d/%s", r.Limit, r.Period.String())
	}

	return m, nil
}
