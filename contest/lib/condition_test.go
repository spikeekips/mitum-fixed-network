// +build mongodb

package contestlib

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type testCondition struct {
	suite.Suite
}

func (t *testCondition) TestBSONM() {
	y := `
query: >
    {"_node": "n0", "a.b": "ab", "c.d": "cd"}
`

	var cm *Condition

	t.NoError(yaml.Unmarshal([]byte(y), &cm))
	t.NoError(cm.IsValid(nil))

	t.Equal("n0", cm.Query()["_node"])
	t.Equal("ab", cm.Query()["a.b"])
	t.Equal("cd", cm.Query()["c.d"])

	b, err := yaml.Marshal(cm)
	t.NoError(err)

	t.Contains(string(b), `{"_node": "n0", "a.b": "ab", "c.d": "cd"}`)
}

func TestCondition(t *testing.T) {
	suite.Run(t, new(testCondition))
}
