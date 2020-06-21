package contestlib

import (
	"bytes"
	"html/template"
	"regexp"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

var (
	reTemplateAssignStringFormat = `\{\{\.[a-zA-Z0-9_][a-zA-Z0-9_]*\}\}`
	reTemplateAssignString       = regexp.MustCompile(reTemplateAssignStringFormat)
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
	Register     []*ConditionRegister
	query        bson.M
	tmpl         *template.Template
	action       ConditionAction
}

func (dc *Condition) String() string {
	return jsonenc.ToString(dc.query)
}

func (dc *Condition) IsValid([]byte) error {
	if err := dc.IfError.IsValid(nil); err != nil {
		return err
	}

	for _, r := range dc.Register {
		if err := r.IsValid(nil); err != nil {
			return err
		}
	}

	n := reTemplateAssignString.ReplaceAll([]byte(dc.QueryString), []byte("1"))
	if err := jsonenc.Unmarshal(n, &bson.M{}); err != nil {
		return err
	}

	if t, err := template.New("query").Parse(dc.QueryString); err != nil {
		return err
	} else {
		dc.tmpl = t
	}

	return nil
}

func (dc *Condition) Action() ConditionAction {
	return dc.action
}

func (dc *Condition) FormatQuery(m map[string]interface{}) (bson.M, error) {
	if dc.query != nil {
		return dc.query, nil
	}

	var bf bytes.Buffer
	if err := dc.tmpl.Execute(&bf, m); err != nil {
		return nil, err
	}

	var q bson.M
	if err := bson.UnmarshalExtJSON(bf.Bytes(), false, &q); err != nil {
		return nil, err
	}

	dc.query = q

	return dc.query, nil
}
