package pm

import (
	"context"
	"strings"

	"golang.org/x/xerrors"
)

type ProcessFunc func(context.Context) (context.Context, error)

type Process interface {
	Name() string
	Requires() []string
	Disabled() bool
	Run(context.Context) (context.Context, error)
}

type BaseProcess struct {
	name     string
	requires []string
	f        ProcessFunc
}

func NewDisabledProcess(process Process) BaseProcess {
	return BaseProcess{
		name:     process.Name(),
		requires: process.Requires(),
		f:        nil,
	}
}

func NewProcess(name string, requires []string, f ProcessFunc) (BaseProcess, error) {
	name = strings.TrimSpace(name)
	if len(name) < 1 {
		return BaseProcess{}, xerrors.Errorf("empty name found")
	}

	rs := requires[:0]
	for _, r := range requires {
		switch s := strings.TrimSpace(r); {
		case len(s) < 1:
			return BaseProcess{}, xerrors.Errorf("empty require found")
		case name == r:
			return BaseProcess{}, xerrors.Errorf("same name found in requires, %q", r)
		default:
			rs = append(rs, r)
		}
	}

	return BaseProcess{
		name:     name,
		requires: rs,
		f:        f,
	}, nil
}

func (pr BaseProcess) Name() string {
	return pr.name
}

func (pr BaseProcess) Requires() []string {
	return pr.requires
}

func (pr BaseProcess) Run(ctx context.Context) (context.Context, error) {
	if pr.f == nil {
		return ctx, nil
	}

	return pr.f(ctx)
}

func (pr BaseProcess) Disabled() bool {
	return pr.f == nil
}

func EmptyProcessFunc(ctx context.Context) (context.Context, error) {
	return ctx, nil
}
