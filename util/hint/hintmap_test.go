package hint

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type testHintmap struct {
	suite.Suite
}

func (t *testHintmap) TestAdd() {
	ty := Type{0xff, 0x70}
	t.NoError(registerType(ty, "findmeff70"))

	sh := newSomethingHinted(ty, "2019.10-alpha", 10)

	hm := NewHintmap()
	err := hm.Add(sh, 33)
	t.NoError(err)

	i, found := hm.Get(sh) // same hint
	t.True(found)
	t.Equal(33, i)

	shl := newSomethingHinted(ty, "2019.9-alpha", 10)
	i, found = hm.Get(shl) // lower version
	t.True(found)
	t.Equal(33, i)
}

func TestHintmap(t *testing.T) {
	suite.Run(t, new(testHintmap))
}
