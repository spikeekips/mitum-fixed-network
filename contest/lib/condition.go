package contestlib

import (
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
)

type ConditionActionIfError uint8

const (
	ConditionActionIfErrorIgnore ConditionActionIfError = iota
	ConditionActionIfErrorStopContest
)

func (st ConditionActionIfError) String() string {
	switch st {
	case ConditionActionIfErrorIgnore:
		return "ignore"
	case ConditionActionIfErrorStopContest:
		return "stop-contest"
	default:
		return "<unknown ConditionActionIfError>"
	}
}

func (st ConditionActionIfError) IsValid([]byte) error {
	switch st {
	case ConditionActionIfErrorIgnore,
		ConditionActionIfErrorStopContest:
		return nil
	}

	return xerrors.Errorf("invalid IfError found; %d", st)
}

func (st ConditionActionIfError) MarshalYAML() (interface{}, error) {
	return st.String(), nil
}

func (st *ConditionActionIfError) UnmarshalYAML(value *yaml.Node) error {
	var s string
	if err := value.Decode(&s); err != nil {
		return err
	}

	switch s {
	case "", "ignore":
		*st = ConditionActionIfErrorIgnore

		return nil
	case "stop-contest":
		*st = ConditionActionIfErrorStopContest

		return nil
	default:
		return xerrors.Errorf("unknown ConditionActionIfError: %s", s)
	}
}

type Condition struct {
	QueryString  string                 `yaml:"query"`
	ActionString string                 `yaml:"action"`
	Args         []string               `yaml:"args"`
	IfError      ConditionActionIfError `yaml:"if-error"`
	query        bson.M
	action       ConditionAction
	lastID       interface{}
}

func (dc *Condition) String() string {
	return dc.QueryString
}

func (dc *Condition) Query() bson.M {
	if dc.lastID != nil {
		dc.query["_id"] = bson.M{"$gt": dc.lastID}
	}

	return dc.query
}

func (dc *Condition) IsValid([]byte) error {
	if err := dc.IfError.IsValid(nil); err != nil {
		return err
	}

	var m bson.M
	if err := jsonenc.Unmarshal([]byte(dc.QueryString), &m); err != nil {
		return err
	}

	dc.query = m

	return nil
}

func (dc *Condition) Action() ConditionAction {
	return dc.action
}

func (dc *Condition) SetLastID(id interface{}) {
	dc.lastID = id
}
