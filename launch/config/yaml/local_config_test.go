package yamlconfig

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type testLocalConfig struct {
	suite.Suite
}

func (t *testLocalConfig) TestEmpty() {
	y := ""

	var n LocalConfig
	err := yaml.Unmarshal([]byte(y), &n)
	t.NoError(err)

	t.Nil(n.SyncInterval)
}

func (t *testLocalConfig) TestEmptySyncInterval() {
	y := `
sync-interval:
`

	var n LocalConfig
	err := yaml.Unmarshal([]byte(y), &n)
	t.NoError(err)

	t.Nil(n.SyncInterval)
}

func (t *testLocalConfig) TestSyncInterval() {
	y := `
sync-interval: 3s
`

	var n LocalConfig
	err := yaml.Unmarshal([]byte(y), &n)
	t.NoError(err)

	t.Equal("3s", *n.SyncInterval)
}

func TestLocalConfig(t *testing.T) {
	suite.Run(t, new(testLocalConfig))
}
