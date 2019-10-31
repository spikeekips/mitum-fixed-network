package contest_config

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

type testSubConfig0 struct {
	Name  string   `yaml:"name"`
	Value []string `yaml:"value"`
}

func (tc testSubConfig0) IsValid() error {
	if len(tc.Value) < 1 {
		return xerrors.Errorf("value should be set at least one")
	}

	return nil
}

func (tc *testSubConfig0) Merge(i interface{}) error {
	ii, ok := i.(testSubConfig0)
	if !ok {
		return xerrors.Errorf("source object is not testSubConfig0 object; %T", i)
	}

	if len(tc.Value) < 1 {
		tc.Value = ii.Value
	}

	return nil
}

type testMainConfig struct {
	AA NameBasedConfig `yaml:"aa"`
	CC testSubConfig   `yaml:"cc"`
}

type testConfigs struct {
	suite.Suite
}

func (t *testConfigs) TestLoad() {
	source := `
aa:
  name: testSubConfig0
  value:
    - value0
    - value1
cc:
  name: testSubConfig
  x: value x
  y: 3
`
	aa := NewNameBasedConfig(map[string]interface{}{
		"testSubConfig0": testSubConfig0{},
	})
	main := testMainConfig{AA: aa}

	err := yaml.Unmarshal([]byte(source), &main)
	t.NoError(err)

	_, ok := main.AA.Instance().(*testSubConfig0)
	t.True(ok)
	t.IsType(testSubConfig{}, main.CC)

	t.Equal("testSubConfig0", main.AA.Instance().(*testSubConfig0).Name)
	t.Equal([]string{"value0", "value1"}, main.AA.Instance().(*testSubConfig0).Value)
	t.Equal("testSubConfig", main.CC.Name)
	t.Equal("value x", main.CC.ValueX)
	t.Equal(3, main.CC.ValueY)
}

func (t *testConfigs) TestLoadWithoutTypes() {
	source := `
aa:
  name: testSubConfig0
  value:
    - value0
    - value1
cc:
  name: testSubConfig
  x: value x
  y: 3
`
	var main testMainConfig

	err := yaml.Unmarshal([]byte(source), &main)
	t.Contains(err.Error(), "given module not found")
}

func TestConfigs(t *testing.T) {
	suite.Run(t, new(testConfigs))
}
