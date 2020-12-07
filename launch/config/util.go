package config

import (
	"net/url"
	"reflect"
	"strings"
	"time"

	"golang.org/x/xerrors"
)

func ParseURLString(s string, allowEmpty bool) (*url.URL, error) { // nolint:unparam
	if s = strings.TrimSpace(s); len(s) < 1 {
		if !allowEmpty {
			return nil, xerrors.Errorf("empty string")
		}

		return nil, nil
	}

	return url.Parse(s)
}

func parseTimeDuration(s string, allowEmpty bool) (time.Duration, error) { // nolint:unparam
	if s = strings.TrimSpace(s); len(s) < 1 {
		if !allowEmpty {
			return 0, xerrors.Errorf("empty string")
		}

		return 0, nil
	} else if t, err := time.ParseDuration(s); err != nil {
		return 0, err
	} else {
		return t, nil
	}
}

func IfNotNil(v interface{}, f func() error) error {
	if reflect.ValueOf(v).IsNil() {
		return nil
	}

	return f()
}

func ParseMap(m map[string]interface{}, key string, allowEmpty bool) (map[string]interface{}, error) {
	if i, found := m[key]; !found || i == nil {
		if !allowEmpty {
			return nil, xerrors.Errorf("empty map")
		} else {
			return nil, nil
		}
	} else if n, ok := i.(map[string]interface{}); !ok {
		return nil, xerrors.Errorf("invalid map, %T found", i)
	} else {
		return n, nil
	}
}

func ParseType(m map[string]interface{}, allowEmpty bool) (string, error) {
	if i, found := m["type"]; !found || i == nil {
		if !allowEmpty {
			return "", xerrors.Errorf("type is missing")
		} else {
			return "", nil
		}
	} else if s, ok := i.(string); !ok {
		return "", xerrors.Errorf("invalid type, %T found", i)
	} else {
		return s, nil
	}
}
