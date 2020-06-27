package hint

import (
	"fmt"

	"github.com/spikeekips/mitum/util"
)

func (t *testHint) TestMarshalJSON() {
	ty := Type{0xff, 0xf0}
	v := util.Version("0.1")

	_ = registerType(ty, "0xfff0")

	h, err := NewHint(ty, v)
	t.NoError(err)

	b, err := util.JSON.Marshal(h)
	t.NoError(err)

	var m map[string]interface{}
	t.NoError(util.JSON.Unmarshal(b, &m))

	t.Contains(fmt.Sprintf("%v", m["type"]), h.Type().String())
	t.Equal(h.Version().String(), m["version"])

	// unmarshal
	var uh Hint
	t.NoError(util.JSON.Unmarshal(b, &uh))
	t.Equal(h.Type(), uh.Type())
	t.Equal(h.Version(), uh.Version())
}
