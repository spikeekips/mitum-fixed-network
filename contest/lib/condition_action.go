package contestlib

import (
	"github.com/spikeekips/mitum/util/isvalid"
	"github.com/spikeekips/mitum/util/logging"
)

type ConditionActionLoader = func([]string) (func() error, error)

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
	f       func() error
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
	err := ca.f()
	if err != nil {
		ca.Log().Error().Err(err).Msg("something wrong")
	} else {
		ca.Log().Verbose().Msg("run")
	}

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
