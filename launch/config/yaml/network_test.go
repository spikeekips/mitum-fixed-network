package yamlconfig

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type testNetwork struct {
	suite.Suite
}

func (t *testNetwork) TestNodeNetwork() {
	y := `
url: https://local:54321
`

	var n NodeNetwork
	err := yaml.Unmarshal([]byte(y), &n)
	t.NoError(err)

	t.Equal("https://local:54321", *n.URL)
}

func (t *testNetwork) TestEmpty() {
	y := ""

	var n NodeNetwork
	err := yaml.Unmarshal([]byte(y), &n)
	t.NoError(err)

	t.True(n.URL == nil)
}

func (t *testNetwork) TestLocalNetwork() {
	y := `
url: https://local:54321
bind: quic://0.0.0.0:54321
cache: dummy://
`

	var n LocalNetwork
	err := yaml.Unmarshal([]byte(y), &n)
	t.NoError(err)

	t.Equal("https://local:54321", *n.URL)
	t.Equal("quic://0.0.0.0:54321", *n.Bind)
	t.Equal("dummy://", *n.Cache)
}

func (t *testNetwork) TestLocalNetworkEmpty() {
	y := ""

	var n LocalNetwork
	err := yaml.Unmarshal([]byte(y), &n)
	t.NoError(err)

	t.True(n.URL == nil)
	t.True(n.Bind == nil)
}

func TestNetwork(t *testing.T) {
	suite.Run(t, new(testNetwork))
}
