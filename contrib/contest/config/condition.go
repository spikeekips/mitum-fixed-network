package contest_config

import (
	"reflect"

	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/contrib/contest/condition"
)

type ConditionAction struct {
	Action string      `yaml:"action"`
	Value  interface{} `yaml:"value,omitempty"`
}

type ActionCondition struct {
	Condition     string            `yaml:"condition"`
	Actions       []ConditionAction `yaml:"actions"`
	actionChecker condition.ActionChecker
}

func (ca *ActionCondition) IsValid() error {
	var conditionChecker condition.ConditionChecker
	if len(ca.Condition) < 1 {
		return xerrors.Errorf("empty `condition`")
	} else {
		if cc, err := condition.NewConditionChecker(ca.Condition); err != nil {
			return err
		} else {
			conditionChecker = cc
		}
	}

	if len(ca.Actions) < 1 {
		return xerrors.Errorf("empty `actions`")
	}

	var actions []condition.Action
	for _, action := range ca.Actions {
		if len(action.Action) < 1 {
			return xerrors.Errorf("empty `action`")
		}

		var hint reflect.Kind
		var values []interface{}
		if action.Value != nil {
			if sl, ok := action.Value.([]interface{}); !ok {
				hint = reflect.TypeOf(action.Value).Kind()
				values = []interface{}{action.Value}
			} else if len(sl) > 0 {
				var vt *reflect.Kind
				for _, i := range sl {
					ik := reflect.TypeOf(i).Kind()
					if vt == nil {
						vt = &ik
						continue
					} else if ik == *vt {
						continue
					}

					return xerrors.Errorf(
						"invalid value type found; values types in list values should be same; expected=%q given=%q value=%v",
						(*vt).String(), ik.String(), i,
					)
				}

				hint = *vt
				values = sl
			}
		}
		actions = append(
			actions,
			condition.NewAction(
				action.Action,
				condition.NewActionValue(values, hint),
			),
		)
	}

	ca.actionChecker = condition.NewActionChecker(
		conditionChecker,
		actions...,
	)

	return nil
}

func (ca *ActionCondition) ActionChecker() condition.ActionChecker {
	return ca.actionChecker
}
