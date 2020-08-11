package contestlib

import (
	"html/template"
	"regexp"
	"strings"

	jsonenc "github.com/spikeekips/mitum/util/encoder/json"
	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/xerrors"
	"gopkg.in/yaml.v3"
)

var (
	reConditionQueryStringFormat = `\{\{[\s]*[a-zA-Z0-9_\.][a-zA-Z0-9_\.]*[\s]*\}\}`
	reConditionQueryString       = regexp.MustCompile(reConditionQueryStringFormat)
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
	ColString    string                 `yaml:"col"`
	Register     []*ConditionRegister
	query        bson.M
	tmpl         *template.Template
	action       ConditionAction
	col          [2]string
}

func (dc *Condition) String() string {
	return jsonenc.ToString(dc.query)
}

func (dc *Condition) IsValid([]byte) error {
	if err := dc.IfError.IsValid(nil); err != nil {
		return err
	}

	dc.ActionString = strings.TrimSpace(dc.ActionString)

	for _, r := range dc.Register {
		if err := r.IsValid(nil); err != nil {
			return err
		}
	}

	if !reConditionQueryString.Match([]byte(dc.QueryString)) {
		var q bson.M
		if err := jsonenc.Unmarshal([]byte(dc.QueryString), &q); err != nil {
			return err
		}

		dc.query = q
	} else {
		n := reConditionQueryString.ReplaceAll([]byte(dc.QueryString), []byte("1"))
		if err := jsonenc.Unmarshal(n, &bson.M{}); err != nil {
			return err
		}

		if t, err := template.New("query").Parse(dc.QueryString); err != nil {
			return err
		} else {
			dc.tmpl = t
		}
	}

	if s := strings.TrimSpace(dc.ColString); len(s) < 1 {
		dc.col = [2]string{}
	} else if l := strings.Split(s, "."); len(l) < 2 {
		return xerrors.Errorf(`invalid col string, "<db>.<collection>"`)
	} else {
		dc.col = [2]string{l[0], strings.Join(l[1:], ".")}
	}

	return nil
}

func (dc *Condition) Action() ConditionAction {
	return dc.action
}

func (dc *Condition) FormatQuery(vars *Vars) (bson.M, error) {
	if dc.query != nil {
		return dc.query, nil
	}

	var q bson.M
	if err := bson.UnmarshalExtJSON([]byte(vars.Format(dc.tmpl)), false, &q); err != nil {
		return nil, err
	}

	dc.query = q

	return dc.query, nil
}

func (dc *Condition) DB() string {
	if dc.col[0] == "" {
		return ""
	}

	return dc.col[0]
}

func (dc *Condition) Collection() string {
	if dc.col[1] == "" {
		return EvenCollection
	}

	return dc.col[1]
}
