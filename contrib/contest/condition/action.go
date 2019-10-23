package condition

import "reflect"

type ActionValue struct {
	value interface{}
	kind  reflect.Kind
}

func NewActionValue(value interface{}, kind reflect.Kind) ActionValue {
	return ActionValue{value: value, kind: kind}
}

func (av ActionValue) Value() interface{} {
	return av.value
}

func (av ActionValue) Hint() reflect.Kind {
	return av.kind
}

type Action struct {
	checker ConditionChecker
	action  string
	value   ActionValue
}

func NewAction(checker ConditionChecker, action string, value ActionValue) Action {
	return Action{checker: checker, action: action, value: value}
}

func NewActionWithoutValue(checker ConditionChecker, action string) Action {
	return Action{checker: checker, action: action}
}

func (ac Action) Checker() ConditionChecker {
	return ac.checker
}

func (ac Action) Action() string {
	return ac.action
}

func (ac Action) Value() ActionValue {
	return ac.value
}
