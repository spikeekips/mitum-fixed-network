package main

import (
	"fmt"
	"reflect"
	"strings"
)

type Condition interface {
	Hint() reflect.Kind
	String() string
}

type Comparison struct {
	name  string
	op    string
	value []interface{}
	kind  reflect.Kind
}

func NewComparison(name, op string, value []interface{}, kind reflect.Kind) Comparison {
	return Comparison{name: name, op: op, value: value, kind: kind}
}

func (bc Comparison) Name() string {
	return bc.name
}

func (bc Comparison) Op() string {
	return bc.op
}

func (bc Comparison) Value() interface{} {
	switch bc.op {
	case "in", "not in":
		return bc.value
	}

	if len(bc.value) < 1 {
		return nil
	}

	return bc.value[0]
}

func (bc Comparison) Hint() reflect.Kind {
	return bc.kind
}

func (bc Comparison) String() string {
	var vs []string
	for _, v := range bc.value {
		vs = append(vs, fmt.Sprintf("%v", v))
	}

	return fmt.Sprintf("(%s %s [%s])", bc.name, bc.op, strings.Join(vs, ","))
}

type JointConditions struct {
	op         string
	conditions []Condition
}

func NewJointConditions(op string, conditions ...Condition) JointConditions {
	return JointConditions{op: op, conditions: conditions}
}

func (bc JointConditions) Add(conditions ...Condition) JointConditions {
	if len(bc.conditions) < 1 && len(conditions) == 1 {
		switch t := conditions[0].(type) {
		case JointConditions:
			if len(bc.op) < 1 || bc.op == t.op {
				return t
			}
		}
	}
	bc.conditions = append(bc.conditions, conditions...)

	return bc
}

func (bc JointConditions) Conditions() []Condition {
	return bc.conditions
}

func (bc JointConditions) Op() string {
	return bc.op
}

func (bc JointConditions) Hint() reflect.Kind {
	return reflect.Invalid
}

func (bc JointConditions) String() string {
	var l []string
	for _, c := range bc.conditions {
		l = append(l, c.String())
	}

	op := bc.op
	if len(op) < 1 {
		op = "and"
	}

	return fmt.Sprintf("(%s:%s)", op, strings.Join(l, ", "))
}
