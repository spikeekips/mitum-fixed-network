package hint

import (
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
)

func (t *testHint) TestMarshalBSON() {
	ty := Type{0xff, 0x36}
	v := Version("0.1")

	_ = registerType(ty, "0xff36")

	h, err := NewHint(ty, v)
	t.NoError(err)

	b, err := bson.Marshal(h)
	t.NoError(err)

	var m bson.M
	t.NoError(bson.Unmarshal(b, &m))

	t.Contains(fmt.Sprintf("%v", m["type"]), h.Type().String())
	t.Equal(h.Version().String(), m["version"])

	// unmarshal
	var uh Hint
	t.NoError(bson.Unmarshal(b, &uh))
	t.Equal(h.Type(), uh.Type())
	t.Equal(h.Version(), uh.Version())
}
