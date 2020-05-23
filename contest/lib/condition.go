package contestlib

import (
	"go.mongodb.org/mongo-driver/bson"

	jsonencoder "github.com/spikeekips/mitum/util/encoder/json"
)

type Condition struct {
	QueryString string `yaml:"query"`
	query       bson.M
}

func (dc *Condition) String() string {
	return dc.QueryString
}

func (dc *Condition) Query() bson.M {
	return dc.query
}

func (dc *Condition) IsValid([]byte) error {
	var m bson.M
	if err := jsonencoder.Unmarshal([]byte(dc.QueryString), &m); err != nil {
		return err
	}

	dc.query = m

	return nil
}
