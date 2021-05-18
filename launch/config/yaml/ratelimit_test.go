package yamlconfig

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type testRateLimit struct {
	suite.Suite
}

func (t *testRateLimit) TestNew() {
	y := `
preset:
  suffrage:
     default: 600/2m # default endpoint
     blockdata-maps: 700/3m
     blockdata: 80/4m

  world:
     default: 500/2m # default endpoint
     new-seal: 300/2m
     blockdata: 60/1m

192.168.1.0/24:
  preset: suffrage
  new-seal: 222/1s
  blockdata-maps: 333/2s

192.168.2.0/24:
  preset: world
  blockdata-maps: 444/1m

192.168.3.0:
  preset: world
`

	var m *RateLimit

	err := yaml.Unmarshal([]byte(y), &m)
	t.NoError(err)

	t.NotNil(m.preset)
	t.NotNil(m.preset["suffrage"])
	t.NotNil(m.preset["world"])

	t.NotNil(m.preset["suffrage"]["default"])
	t.NotNil(m.preset["suffrage"]["blockdata-maps"])
	t.NotNil(m.preset["suffrage"]["blockdata"])

	t.NotNil(m.preset["world"]["default"])
	t.NotNil(m.preset["world"]["new-seal"])
	t.NotNil(m.preset["world"]["blockdata"])

	t.Equal(int64(600), m.preset["suffrage"]["default"].Limit)
	t.Equal(time.Minute*2, m.preset["suffrage"]["default"].Period)
	t.Equal(int64(700), m.preset["suffrage"]["blockdata-maps"].Limit)
	t.Equal(time.Minute*3, m.preset["suffrage"]["blockdata-maps"].Period)
	t.Equal(int64(80), m.preset["suffrage"]["blockdata"].Limit)
	t.Equal(time.Minute*4, m.preset["suffrage"]["blockdata"].Period)

	t.Equal(int64(300), m.preset["world"]["new-seal"].Limit)
	t.Equal(time.Minute*2, m.preset["world"]["new-seal"].Period)
	t.Equal(int64(60), m.preset["world"]["blockdata"].Limit)
	t.Equal(time.Minute, m.preset["world"]["blockdata"].Period)

	t.NotNil(m.rules)
	t.Equal(3, len(m.rules))
	t.NotNil(m.rules[0])
	t.NotNil(m.rules[1])
	t.NotNil(m.rules[2])

	t.Equal("192.168.1.0/24", m.rules[0].target)
	t.Equal("suffrage", m.rules[0].preset)
	t.Equal(int64(222), m.rules[0].rules["new-seal"].Limit)
	t.Equal(time.Second, m.rules[0].rules["new-seal"].Period)
	t.Equal(int64(333), m.rules[0].rules["blockdata-maps"].Limit)
	t.Equal(time.Second*2, m.rules[0].rules["blockdata-maps"].Period)

	t.Equal("192.168.2.0/24", m.rules[1].target)
	t.Equal("world", m.rules[1].preset)
	t.Equal(int64(444), m.rules[1].rules["blockdata-maps"].Limit)
	t.Equal(time.Minute, m.rules[1].rules["blockdata-maps"].Period)

	t.Equal("192.168.3.0", m.rules[2].target)
	t.Equal("world", m.rules[2].preset)
	t.Equal(0, len(m.rules[2].rules))
}

func (t *testRateLimit) TestEmptyPresetInSet() {
	y := `
preset:
  suffrage:
     default: 600/2m # default endpoint
     blockdata-maps: 700/3m
     blockdata: 80/4m

  world:
     default: 500/2m # default endpoint
     new-seal: 300/2m
     blockdata: 60/1m

192.168.1.0/24:
  preset: suffrage
  new-seal: 222/1s
  blockdata-maps: 333/2s

192.168.2.0/24:
  blockdata-maps: 444/1m
`

	var m *RateLimit

	err := yaml.Unmarshal([]byte(y), &m)
	t.NoError(err)
}

func (t *testRateLimit) TestEmptySets() {
	y := `
preset:
  suffrage:
     default: 600/2m # default endpoint
     blockdata-maps: 700/3m
     blockdata: 80/4m

  world:
     default: 500/2m # default endpoint
     new-seal: 300/2m
     blockdata: 60/1m
`

	var m *RateLimit

	err := yaml.Unmarshal([]byte(y), &m)
	t.NoError(err)

	t.Empty(m.rules)
}

func (t *testRateLimit) TestEmptySet() {
	y := `
preset:
  suffrage:
     default: 600/2m # default endpoint
     blockdata-maps: 700/3m
     blockdata: 80/4m

  world:
     default: 500/2m # default endpoint
     new-seal: 300/2m
     blockdata: 60/1m

192.168.1.0/24:
  preset: suffrage
  new-seal: 222/1s
  blockdata-maps: 333/2s

192.168.2.0/24:
`

	var m *RateLimit

	err := yaml.Unmarshal([]byte(y), &m)
	t.NoError(err)

	t.Equal(2, len(m.rules))

	r := m.rules[1]
	t.Equal("192.168.2.0/24", r.target)
	t.Equal("", r.preset)
	t.Empty(r.rules)
}

func (t *testRateLimit) TestNoDigitDuration() {
	y := `
192.168.1.0/24:
  new-seal: 333/ms
`

	var m *RateLimit

	err := yaml.Unmarshal([]byte(y), &m)
	t.NoError(err)

	t.Equal("192.168.1.0/24", m.rules[0].target)
	t.Equal(int64(333), m.rules[0].rules["new-seal"].Limit)
	t.Equal(time.Millisecond, m.rules[0].rules["new-seal"].Period)
}

func (t *testRateLimit) TestCache() {
	y := `
cache: gcache:?type=lru&size=33&expire=44s
`

	var m *RateLimit

	err := yaml.Unmarshal([]byte(y), &m)
	t.NoError(err)

	t.NotNil(m.cache)
	t.Equal("gcache:?type=lru&size=33&expire=44s", *m.cache)
}

func TestRateLimit(t *testing.T) {
	suite.Run(t, new(testRateLimit))
}
