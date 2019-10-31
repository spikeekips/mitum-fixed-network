package contest_config

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
)

type testActionCondition struct {
	suite.Suite
}

func (t *testActionCondition) TestNew() {
	conditionString := "a=1"
	actionString := "do-something"
	valueString := 1
	source := fmt.Sprintf(`
condition: %s
actions:
  - action: %s
    value: %v
`, conditionString, actionString, valueString)

	var ca ActionCondition
	err := yaml.Unmarshal([]byte(source), &ca)
	t.NoError(err)

	t.Equal(conditionString, ca.Condition)
	t.Equal(actionString, ca.Actions[0].Action)
	t.Equal(valueString, ca.Actions[0].Value)
}

func (t *testActionCondition) TestIsValid() {
	conditionString := "a=1"
	actionString := "do-something"
	valueString := 1
	source := fmt.Sprintf(`
condition: %s
actions:
  - action: %s
    value: %v
`, conditionString, actionString, valueString)

	var ca ActionCondition
	err := yaml.Unmarshal([]byte(source), &ca)
	t.NoError(err)

	err = ca.IsValid()
	t.NoError(err)

	t.Equal(conditionString, ca.Condition)
	t.Equal(actionString, ca.Actions[0].Action)
	t.Equal(valueString, ca.Actions[0].Value)

	t.NotEmpty(ca.ActionChecker().Actions())
	t.Equal(reflect.Int, ca.ActionChecker().Actions()[0].Value().Hint())
}

func (t *testActionCondition) TestIsValidCases() {
	cases := []struct {
		name      string
		source    string
		err       string
		checkFunc func(ActionCondition) error
	}{
		{
			name: "filled",
			source: `
condition: a = 1
actions:
  - action: do-somthing
    value: 2`,
		},
		{
			name: "missing condition",
			source: `
actions:
  - action: do-somthing
    value: 2`,
			err: "empty `condition`",
		},
		{
			name: "missing action",
			source: `
condition: a = 1
actions:
  - value: 2`,
			err: "empty `action`",
		},
		{
			name: "missing value",
			source: `
condition: a = 1
actions:
  - action: do-somthing`,
		},
		{
			name: "invalid condition query",
			source: `
condition: a == 1"
actions:
  - action: do-somthing
    value: 2`,
			err: "syntax error at position",
		},
		{
			name: "int value list",
			source: `
condition: a = 1
actions:
  - action: do-somthing
    value:
    - 0
    - 1
    - 2
`,
			checkFunc: func(ca ActionCondition) error {
				t.Equal([]interface{}{0, 1, 2}, ca.Actions[0].Value)
				t.Equal(reflect.Int, ca.ActionChecker().Actions()[0].Value().Hint())

				return nil
			},
		},
		{
			name: "string value list",
			source: `
condition: a = 1
actions:
  - action: do-somthing
    value:
    - "a"
    - "b"
    - "c"
`,
			checkFunc: func(ca ActionCondition) error {
				t.Equal([]interface{}{"a", "b", "c"}, ca.Actions[0].Value)
				t.Equal(reflect.String, ca.ActionChecker().Actions()[0].Value().Hint())

				return nil
			},
		},
		{
			name: "mixed value list",
			source: `
condition: a = 1
actions:
  - action: do-somthing
    value:
    - "a"
    - 1
    - "c"
`,
			err: "invalid value type found",
		},
	}

	for i, c := range cases {
		i := i
		c := c
		t.T().Run(
			c.name,
			func(*testing.T) {
				var ca ActionCondition
				err := yaml.Unmarshal([]byte(c.source), &ca)
				t.NoError(err)

				err = ca.IsValid()
				if len(c.err) < 1 {
					t.NoError(err, "not expected error: %q, %d: %v", err, i, c.name)
				} else {
					t.Contains(err.Error(), c.err, "%d: %v; %v != %v", i, c.name, c.err, err)
				}

				if c.checkFunc != nil {
					err = c.checkFunc(ca)
					t.NoError(err, "not expected error in checkFunc: %q, %d: %v", err, i, c.name)
				}
			},
		)
	}
}

func TestActionCondition(t *testing.T) {
	suite.Run(t, new(testActionCondition))
}
