package condition

import (
	"reflect"
	"sync"
)

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

func NewActionCheckerWithoutValue(checker ConditionChecker, action string) ActionChecker {
	return ActionChecker{checker: checker, actions: []Action{Action{action: action}}}
}

func NewActionChecker(checker ConditionChecker, actions ...Action) ActionChecker {
	return ActionChecker{checker: checker, actions: actions}
}

func (ac ActionChecker) Checker() ConditionChecker {
	return ac.checker
}

func (ac ActionChecker) Actions() []Action {
	return ac.actions
}

type MultiActionCheckers struct {
	sync.RWMutex
	actionCheckers []ActionChecker
	as             []ActionChecker
}

func NewMultiActionCheckers(checkers []ActionChecker) *MultiActionCheckers {
	for _, c := range checkers {
		var actions []Action
		for _, action := range c.Actions() {
			if len(action.Value().Value()) < 1 {
				continue
			} else if action.Value().Hint() != reflect.Func {
				continue
			}
			actions = append(actions, action)
		}
		c.actions = actions
	}

	return &MultiActionCheckers{
		actionCheckers: checkers,
		as:             checkers,
	}
}

func (ma *MultiActionCheckers) actives() []ActionChecker {
	ma.RLock()
	defer ma.RUnlock()

	return ma.as
}

func (ma *MultiActionCheckers) Check(o LogItem) bool {
	if len(ma.actives()) < 1 {
		return false
	}

	var found bool
	var actives []ActionChecker
	for _, c := range ma.actives() {
		if !c.Checker().Check(o) {
			actives = append(actives, c)
			continue
		} else if !found {
			found = true
		}

		for _, action := range c.Actions() {
			go action.Value().Value()[0].(func())()
		}
	}

	ma.Lock()
	ma.as = actives
	ma.Unlock()

	return found
}
