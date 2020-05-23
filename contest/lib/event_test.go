package contestlib

import (
	"testing"

	"github.com/stretchr/testify/suite"
	"go.mongodb.org/mongo-driver/bson"
)

type testEvent struct {
	suite.Suite
}

func (t *testEvent) TestPreserveFields() {
	b := `{
  "level": "info",
  "module": "consensus-states",
  "handler": 33,
  "t": "2020-05-21T08:19:45.76229515Z",
  "context": {
    "from": "BOOTING",
    "to": "JOINING"
  },
  "m": "activated: JOINING"
}`

	var r bson.Raw
	t.NoError(bson.UnmarshalExtJSON([]byte(b), true, &r))
	t.NoError(r.Validate())

	t.Equal("info", r.Lookup("level").StringValue())
	t.Equal("consensus-states", r.Lookup("module").StringValue())
	t.Equal(int32(33), r.Lookup("handler").Int32())

	t.Equal("BOOTING", r.Lookup("context").Document().Lookup("from").StringValue())
	t.Equal("JOINING", r.Lookup("context").Document().Lookup("to").StringValue())
}

func TestEvent(t *testing.T) {
	suite.Run(t, new(testEvent))
}
