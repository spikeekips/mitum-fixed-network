package hint

import (
	"go.mongodb.org/mongo-driver/bson"

	"github.com/spikeekips/mitum/util"
)

type marshalingHint struct {
	Hint Hint
}

func (t *testHint) TestMarshalBSON() {
	ty := Type{0xff, 0x36}
	v := util.Version("0.1")

	_ = registerType(ty, "0xff36")

	h, err := NewHint(ty, v)
	t.NoError(err)

	b, err := bson.Marshal(marshalingHint{Hint: h})
	t.NoError(err)

	var m marshalingHint
	t.NoError(bson.Unmarshal(b, &m))

	t.True(h.Equal(m.Hint))
}
