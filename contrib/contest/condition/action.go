package condition

import "reflect"

type ActionValue struct {
	value []interface{}
	kind  reflect.Kind
}

func NewActionValue(value []interface{}, kind reflect.Kind) ActionValue {
	return ActionValue{value: value, kind: kind}
}

func (av ActionValue) Value() []interface{} {
	return av.value
}

func (av ActionValue) Hint() reflect.Kind {
	return av.kind
}

type Action struct {
	action string
	value  ActionValue
}

func NewAction(action string, value ActionValue) Action {
	return Action{action: action, value: value}
}

func NewActionWithoutValue(action string) Action {
	return Action{action: action}
}

func (ac Action) Action() string {
	return ac.action
}

func (ac Action) Value() ActionValue {
	return ac.value
}

type ActionChecker struct {
	checker ConditionChecker
	actions []Action
}

func NewActionChecker(checker ConditionChecker, action string, value ActionValue) ActionChecker {
	return ActionChecker{checker: checker, actions: []Action{NewAction(action, value)}}
}

func NewActionCheckerWithoutValue(checker ConditionChecker, action string) ActionChecker {
	return ActionChecker{checker: checker, actions: []Action{Action{action: action}}}
}

func NewActionChecker0(checker ConditionChecker, actions ...Action) ActionChecker {
	return ActionChecker{checker: checker, actions: actions}
}

func (ac ActionChecker) Checker() ConditionChecker {
	return ac.checker
}

func (ac ActionChecker) Actions() []Action {
	return ac.actions
}
