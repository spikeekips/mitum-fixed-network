package main

import (
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
