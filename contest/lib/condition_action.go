package contestlib

import (
	"html/template"
	"strings"

	"github.com/spikeekips/mitum/util"
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type ConditionActionLoader = func([]string) (func(logging.Logger) error, error)

type ConditionAction interface {
	isvalid.IsValider
	Name() string
	Run() error
}

type BaseConditionAction struct {
	*logging.Logging
	name    string
	load    ConditionActionLoader
	args    []string
	f       func(logging.Logger) error
	iferror ConditionActionIfError
}

func NewConditionAction(
	name string,
	load ConditionActionLoader,
	args []string,
	iferror ConditionActionIfError,
) *BaseConditionAction {
	return &BaseConditionAction{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.
				Str("name", name).
				Strs("args", args).
				Str("iferror", iferror.String()).
				Str("module", "condition-action")
		}),
		name:    name,
		load:    load,
		args:    args,
		iferror: iferror,
	}
}

func (ca *BaseConditionAction) IsValid([]byte) error {
	if f, err := ca.load(ca.args); err != nil {
		return err
	} else {
		ca.f = f
	}

	return nil
}

func (ca *BaseConditionAction) Name() string {
	return ca.name
}

func (ca *BaseConditionAction) Run() error {
	err := ca.f(ca.Log())

	switch {
	case err == nil:
		return nil
	case ca.iferror == ConditionActionIfErrorIgnore:
		return nil
	case ca.iferror == ConditionActionIfErrorStopContest:
		return err
	default:
		return err
	}
}

func NewShellConditionActionLoader(vars *Vars, s string) (ConditionActionLoader, error) {
	if len(s) < 1 {
		return nil, xerrors.Errorf("empty command for shell action")
	}

	var tmpl *template.Template
	if t, err := template.New("shell-command").Parse(s); err != nil {
		return nil, err
	} else {
		tmpl = t
	}

	return func([]string) (func(logging.Logger) error, error) {
		return func(log logging.Logger) error {
			c := vars.Format(tmpl)
			if len(c) < 1 {
				return xerrors.Errorf("failed to format command")
			}

			log.Debug().Str("command", c).Msg("run shell command")

			return util.Exec(c)
		}, nil
	}, nil
}

func cutShellCommandString(s string) string {
	if !strings.HasPrefix(s, "$ ") {
		return s
	}

	return s[2:]
}
