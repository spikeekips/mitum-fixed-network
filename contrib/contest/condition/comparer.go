package condition

import (
	"reflect"
	"sort"
)

type CompareType interface {
	Cmp() int
}

type CompareInt struct {
	a int64
	b int64
}

func NewCompareInt(a, b int64) CompareInt {
	return CompareInt{a: a, b: b}
}

func (ci CompareInt) Cmp() int {
	c := ci.a - ci.b
	if c == 0 {
		return 0
	} else if c > 0 {
		return 1
	}
	return -1
}

type CompareFloat struct {
	a float64
	b float64
}

func NewCompareFloat(a, b float64) CompareFloat {
	return CompareFloat{a: a, b: b}
}

func (ci CompareFloat) Cmp() int {
	c := ci.a - ci.b
	if c == 0 {
		return 0
	} else if c > 0 {
		return 1
	}
	return -1
}

type CompareString struct {
	a string
	b string
}

func NewCompareString(a, b string) CompareString {
	return CompareString{a: a, b: b}
}

func (ci CompareString) Cmp() int {
	if ci.a == ci.b {
		return 0
	}

	c := []string{ci.a, ci.b}
	sort.Strings(c)

	if ci.a == c[1] {
		return 1
	}

	return -1
}

type CompareNil struct {
	a interface{}
}

func NewCompareNil(a interface{}) CompareNil {
	return CompareNil{a: a}
}

func (ci CompareNil) Cmp() int {
	if ci.a == nil {
		return 0
	}

	return 1
}

type CompareBool struct {
	a interface{}
	b bool
}

func NewCompareBool(a interface{}, b bool) CompareBool {
	return CompareBool{a: a, b: b}
}

func (cb CompareBool) Cmp() int {
	switch cb.a.(type) {
	case bool:
		if cb.a.(bool) == cb.b {
			return 0
		} else {
			return -1
		}
	case string:
		if len(cb.a.(string)) > 0 {
			if cb.b {
				return 0
			} else {
				return -1
			}
		}
		if cb.b {
			return -1
		} else {
			return 0
		}
	case int, int8, int32, int64, uint, uint8, uint32, uint64:
		if a, _ := convertToInt64(cb.a); a == 0 {
			if cb.b {
				return -1
			} else {
				return 0
			}
		}

		if cb.b {
			return 0
		} else {
			return -1
		}
	case float32, float64:
		if a, _ := convertToFloat64(cb.a); a == 0 {
			if cb.b {
				return -1
			} else {
				return 0
			}
		}

		if cb.b {
			return 0
		} else {
			return -1
		}
	}

	switch reflect.TypeOf(cb.a).Kind() {
	case reflect.Array, reflect.Slice:
		if len(cb.a.([]interface{})) > 0 {
			if cb.b {
				return 0
			} else {
				return -1
			}
		} else {
			if cb.b {
				return -1
			} else {
				return 0
			}
		}
	}

	return -1
}
