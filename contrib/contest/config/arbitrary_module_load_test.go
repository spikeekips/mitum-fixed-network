package contest_config

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

type testSubConfig struct {
	Name   string `yaml:"name"`
	ValueX string `yaml:"x,omitempty"`
	ValueY int    `yaml:"y,omitempty"`
}

func (tc testSubConfig) IsValid() error {
	if len(tc.ValueX) > 10 {
		return xerrors.Errorf("ValueX is too long: %d", len(tc.ValueX))
	}

	return nil
}

func (tc *testSubConfig) Merge(i interface{}) error {
	ii, ok := i.(testSubConfig)
	if !ok {
		return xerrors.Errorf("source object is not testSubConfig object; %T", i)
	}

	if len(tc.ValueX) < 1 {
		tc.ValueX = ii.ValueX
	}
	if tc.ValueY < 1 {
		tc.ValueY = ii.ValueY
	}

	return nil
}

type testArbitraryModuleLoad struct {
	suite.Suite
}

func (t *testArbitraryModuleLoad) TestLoad() {
	source := `
name: testSubConfig
x: value x
y: 3
`
	nc := NewNameBasedConfig(map[string]interface{}{
		"testSubConfig": testSubConfig{},
	})

	err := yaml.Unmarshal([]byte(source), &nc)
	t.NoError(err)
	_, ok := nc.instance.(*testSubConfig)
	t.True(ok)
}

func (t *testArbitraryModuleLoad) TestLoadButUnknownName() {
	source := `
name: what?
x: value x
y: 3
`
	nc := NewNameBasedConfig(map[string]interface{}{
		"testSubConfig": testSubConfig{},
	})

	err := yaml.Unmarshal([]byte(source), &nc)
	t.Contains(err.Error(), "given module not found")
}

func (t *testArbitraryModuleLoad) TestLoadButInvalidFieldType() {
	source := `
name: testSubConfig
x: value x
y: findme
`
	nc := NewNameBasedConfig(map[string]interface{}{
		"testSubConfig": testSubConfig{},
	})

	err := yaml.Unmarshal([]byte(source), &nc)
	t.Contains(err.Error(), "cannot unmarshal")
}

func (t *testArbitraryModuleLoad) TestLoadButFailedInvalid() {
	source := `
name: testSubConfig
x: 01234567891 # over 10
y: 2
`
	nc := NewNameBasedConfig(map[string]interface{}{
		"testSubConfig": testSubConfig{},
	})

	err := yaml.Unmarshal([]byte(source), &nc)
	t.Contains(err.Error(), "too long")
}

func (t *testArbitraryModuleLoad) TestMerge() {
	global := testSubConfig{Name: "global", ValueX: "showme", ValueY: 2}
	source := `
name: testSubConfig
x: findme
`
	nc := NewNameBasedConfig(map[string]interface{}{
		"testSubConfig": testSubConfig{},
	})

	err := yaml.Unmarshal([]byte(source), &nc)
	t.NoError(err)

	sc, ok := nc.Instance().(*testSubConfig)
	t.True(ok)
	t.Implements((*Merger)(nil), sc)

	err = sc.Merge(global)
	t.NoError(err)
	t.Equal("testSubConfig", sc.Name)
	t.Equal("findme", sc.ValueX)
	t.Equal(global.ValueY, sc.ValueY)
}

func TestArbitraryModuleLoad(t *testing.T) {
	suite.Run(t, new(testArbitraryModuleLoad))
}
