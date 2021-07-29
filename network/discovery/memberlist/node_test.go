package memberlist

import (
	"net/url"
	"strings"
	"testing"

	"github.com/spikeekips/mitum/util"
	"github.com/stretchr/testify/suite"
)

func NewFakeNodeMeta(publish string, insecure bool) (NodeMeta, error) {
	var u *url.URL
	if len(publish) > 0 {
		i, err := url.Parse(publish)
		if err != nil {
			return NodeMeta{}, err
		}

		u = i
	}

	meta := NodeMeta{
		publish:  u,
		insecure: insecure,
	}

	meta.b, _ = util.JSON.Marshal(meta)

	return meta, nil
}

type testConnMap struct {
	suite.Suite
}

func (t *testConnMap) TestNew() {
	ma := NewConnInfoMap()
	t.NotNil(ma)
}

func (t *testConnMap) TestAdd() {
	ma := NewConnInfoMap()

	u, _ := url.Parse("https://1.2.3.4/node0")
	ci, err := ma.add(u, true)
	t.NoError(err)
	t.NotEmpty(ci)

	t.Equal("https://1.2.3.4:443/node0", ci.URL().String())
	t.NotEmpty(ci.Address)
	t.True(ci.Insecure())

	uci, found := ma.item(ci.Address)
	t.True(found)
	t.Equal(ci, uci)

	// unknown
	unknownAddr := "1.2.3.4:333"
	_, found = ma.item(unknownAddr)
	t.False(found)
}

func (t *testConnMap) TestAddDefaultPort() {
	ma := NewConnInfoMap()

	u, _ := url.Parse("https://1.2.3.4/node0")
	ci, err := ma.add(u, false)
	t.NoError(err)
	t.NotEmpty(ci)

	t.True(strings.HasSuffix(ci.Address, ":443"))
}

func (t *testConnMap) TestAddSame() {
	ma := NewConnInfoMap()

	u, _ := url.Parse("https://1.2.3.4/node0")
	ci0, err := ma.add(u, false)
	t.NoError(err)
	t.NotEmpty(ci0)

	ci1, err := ma.add(u, false)
	t.NoError(err)
	t.NotEmpty(ci1)

	t.True(ci0.Equal(ci1))
}

func (t *testConnMap) TestNormalizeURL() {
	ma := NewConnInfoMap()

	u0, _ := url.Parse("https://1.2.3.4")
	ci0, err := ma.add(u0, false)
	t.NoError(err)
	t.NotEmpty(ci0)

	// empty or '/' path in url will be removed
	u1, _ := url.Parse("https://1.2.3.4/")
	ci1, err := ma.add(u1, false)
	t.NoError(err)
	t.NotEmpty(ci1)

	t.True(ci0.Equal(ci1))

	uci, found := ma.item(ci1.Address)
	t.True(found)
	t.True(ci0.Equal(uci))

	u2, _ := url.Parse("https://1.2.3.4#showme")
	ci2, err := ma.add(u2, false)
	t.NoError(err)
	t.NotEmpty(ci2)

	t.True(ci0.Equal(ci2))
}

func (t *testConnMap) TestAddSameHostButDifferent() {
	ma := NewConnInfoMap()

	u0, _ := url.Parse("https://1.2.3.4:3001/n0")
	addr0, err := ma.add(u0, false)
	t.NoError(err)
	t.NotEmpty(addr0)

	u1, _ := url.Parse("https://1.2.3.4:3001/n1")
	addr1, err := ma.add(u1, false)
	t.NoError(err)
	t.NotEmpty(addr1)

	t.NotEqual(addr0, addr1)
}

func (t *testConnMap) TestRemove() {
	u, _ := url.Parse("https://1.2.3.4:3001/n0")

	ma := NewConnInfoMap()
	ci, err := ma.add(u, false)
	t.NoError(err)
	t.NotEmpty(ci)

	ma = NewConnInfoMap()

	t.False(ma.remove(ci.Address))

	ci, err = ma.add(u, false)
	t.NoError(err)
	t.NotEmpty(ci)

	uci, found := ma.item(ci.Address)
	t.True(found)
	t.True(ci.Equal(uci))

	t.True(ma.remove(ci.Address))

	_, found = ma.item(ci.Address)
	t.False(found)
}

func TestConnMap(t *testing.T) {
	suite.Run(t, new(testConnMap))
}
