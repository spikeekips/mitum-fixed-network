package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"golang.org/x/xerrors"
)

type ConditionChecker struct {
	condition Condition
}

func NewConditionChecker(query string) (ConditionChecker, error) {
	condition, err := NewConditionParser().Parse(query)
	if err != nil {
		return ConditionChecker{}, err
	}

	return ConditionChecker{condition: condition}, nil
}

func (dc ConditionChecker) Check(o map[string]interface{}) bool {
	return dc.check(dc.condition, o)
}

func (dc ConditionChecker) check(condition Condition, o map[string]interface{}) bool {
	switch c := condition.(type) {
	case Comparison:
		return dc.checkComparison(c, o)
	case JointConditions:
		for _, cd := range c.Conditions() {
			if dc.check(cd, o) {
				if c.Op() == "or" {
					return true
				}
			} else {
				if c.Op() == "and" {
					return false
				}
			}
		}

		return c.Op() == "and"
	}

	return true
}

func (dc ConditionChecker) checkComparison(condition Comparison, o map[string]interface{}) bool {
	v, found := lookup(o, condition.Name())
	if !found {
		return false
	}
	return compare(condition.Op(), v, condition.Value(), condition.Hint())
}

func lookup(o map[string]interface{}, keys string) (interface{}, bool) {
	ts := strings.SplitN(keys, ".", -1)

	return lookupByKeys(o, ts)
}

func lookupByKeys(o map[string]interface{}, keys []string) (interface{}, bool) {
	var found bool
	var f interface{}
	for k, v := range o {
		if k != keys[0] {
			continue
		}

		f = v
		found = true
		break
	}

	if len(keys) == 1 {
		return f, found
	}

	if vv, ok := f.(map[string]interface{}); !ok {
		return nil, false
	} else {
		return lookupByKeys(vv, keys[1:])
	}
}

func indirectToInt(v interface{}) (int64, error) {
	switch reflect.TypeOf(v).Kind() {
	case reflect.Int:
		return int64(v.(int)), nil
	case reflect.Int8:
		return int64(v.(int8)), nil
	case reflect.Int16:
		return int64(v.(int16)), nil
	case reflect.Int32:
		return int64(v.(int32)), nil
	case reflect.Int64:
		return int64(v.(int64)), nil
	case reflect.Uint:
		return int64(v.(uint)), nil
	case reflect.Uint8:
		return int64(v.(uint8)), nil
	case reflect.Uint16:
		return int64(v.(uint16)), nil
	case reflect.Uint32:
		return int64(v.(uint32)), nil
	case reflect.Uint64:
		return int64(v.(uint64)), nil
	default:
		return int64(0), xerrors.Errorf("value is not int; %v", v)
	}
}

func indirectToFloat(v interface{}) (float64, error) {
	k := reflect.TypeOf(v).Kind()
	switch {
	case k == reflect.Float32:
		return float64(v.(float32)), nil
	case k == reflect.Float64:
		return float64(v.(float64)), nil
	case strings.Contains(k.String(), "int"):
		a, err := indirectToInt(v)
		if err != nil {
			return float64(0), err
		}
		return float64(a), nil
	default:
		return float64(0), xerrors.Errorf("value is not float; %v", v)
	}
}

func convertToString(v interface{}) (string, error) {
	switch reflect.TypeOf(v).Kind() {
	case reflect.String:
		return v.(string), nil
	default:
		return fmt.Sprintf("%v", v), nil
	}
}

func convertToInt64(v interface{}) (int64, error) {
	k := reflect.TypeOf(v).Kind()
	switch {
	case k == reflect.String:
		var i int64
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", v)), &i); err != nil {
			return i, err
		}

		return i, nil
	case strings.Contains(k.String(), "int"):
		return indirectToInt(v)
	case strings.Contains(k.String(), "float"):
		a, err := indirectToFloat(v)
		if err != nil {
			return int64(0), err
		}

		return int64(a), nil
	default:
		return int64(0), xerrors.Errorf("not int value type found: %v", k)
	}
}

func convertToFloat64(v interface{}) (float64, error) {
	k := reflect.TypeOf(v).Kind()
	switch {
	case k == reflect.String:
		var i float64
		if err := json.Unmarshal([]byte(fmt.Sprintf("%s", v)), &i); err != nil {
			return i, err
		}

		return i, nil
	case strings.Contains(k.String(), "int"):
		return indirectToFloat(v)
	case strings.Contains(k.String(), "float"):
		return indirectToFloat(v)
	default:
		return float64(0), xerrors.Errorf("not float value type found: %v", k)
	}
}

func compare(op string, a, b interface{}, kind reflect.Kind) bool {
	switch op {
	case "in", "not in":
		rv := reflect.ValueOf(b)
		if rv.Kind() != reflect.Slice {
			return false
		}

		for i := 0; i < rv.Len(); i++ {
			if compare("equal", a, rv.Index(i).Interface(), kind) {
				return op == "in"
			}
		}

		return op != "in"
	case "like", "not like", "regexp", "not regexp":
		ca, err := convertToString(a)
		if err != nil {
			return false
		}

		re, ok := b.(*regexp.Regexp)
		if !ok {
			if re, err = regexp.Compile(b.(string)); err != nil {
				return false
			}
		}

		if len(re.FindAll([]byte(ca), -1)) < 1 {
			return op != "regexp" && op != "like"
		}
		return op == "regexp" || op == "like"
	}

	var ct CompareType
	switch kind {
	case reflect.String:
		if a == nil {
			return false
		}

		ca, err := convertToString(a)
		if err != nil {
			return false
		}
		cb, err := convertToString(b)
		if err != nil {
			return false
		}

		ct = NewCompareString(ca, cb)
	case reflect.Int,
		reflect.Int8,
		reflect.Int16,
		reflect.Int32,
		reflect.Int64,
		reflect.Uint,
		reflect.Uint8,
		reflect.Uint16,
		reflect.Uint32,
		reflect.Uint64:
		if a == nil {
			return false
		}

		ca, err := convertToInt64(a)
		if err != nil {
			return false
		}
		cb, err := indirectToInt(b)
		if err != nil {
			return false
		}
		ct = NewCompareInt(ca, cb)
	case reflect.Float32, reflect.Float64:
		if a == nil {
			return false
		}

		ca, err := convertToFloat64(a)
		if err != nil {
			return false
		}
		cb, err := indirectToFloat(b)
		if err != nil {
			return false
		}
		ct = NewCompareFloat(ca, cb)
	case reflectNilKind:
		ct = NewCompareNil(a)
	}

	cmp := ct.Cmp()
	switch op {
	case "equal", "=":
		return cmp == 0
	case "not equal", "!=":
		return cmp != 0
	case "greater_than", ">":
		return cmp > 0
	case "equal_or_greater_than", ">=":
		return cmp >= 0
	case "lesser_than", "<":
		return cmp < 0
	case "equal_or_lesser_than", "<=":
		return cmp <= 0
	}

	return false
}
