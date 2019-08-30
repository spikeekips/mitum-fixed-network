package main

import (
	"fmt"
	"strings"
)

type ConditionChecker interface {
	Name() string
	Op() string
	Value() []interface{}
	String() string
}

type Comparison struct {
	name  string
	op    string
	value []interface{}
}

func NewComparison(name, op string, value []interface{}) Comparison {
	return Comparison{name: name, op: op, value: value}
}

func (bc Comparison) Name() string {
	return bc.name
}

func (bc Comparison) Op() string {
	return bc.op
}

func (bc Comparison) Value() []interface{} {
	return bc.value
}

func (bc Comparison) String() string {
	var vs []string
	for _, v := range bc.value {
		vs = append(vs, fmt.Sprintf("%v", v))
	}

	return fmt.Sprintf("(%s %s [%s])", bc.name, bc.op, strings.Join(vs, ","))
}

type JointConditions struct {
	op       string
	checkers []ConditionChecker
}

func NewJointConditions(op string, checkers ...ConditionChecker) JointConditions {
	return JointConditions{op: op, checkers: checkers}
}

func (bc JointConditions) Add(checkers ...ConditionChecker) JointConditions {
	if len(bc.checkers) < 1 && len(checkers) == 1 {
		switch t := checkers[0].(type) {
		case JointConditions:
			if len(bc.op) < 1 || bc.op == t.op {
				return t
			}
		}
	}
	bc.checkers = append(bc.checkers, checkers...)

	return bc
}

func (bc JointConditions) Name() string {
	return ""
}

func (bc JointConditions) Op() string {
	return bc.op
}

func (bc JointConditions) Value() []interface{} {
	return nil
}

func (bc JointConditions) String() string {
	var l []string
	for _, c := range bc.checkers {
		l = append(l, c.String())
	}

	op := bc.op
	if len(op) < 1 {
		op = "and"
	}

	return fmt.Sprintf("(%s:%s)", op, strings.Join(l, ", "))
}
