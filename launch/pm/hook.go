package pm

import (
	"context"

	"golang.org/x/xerrors"
)

const (
	HookDirBefore = "before"
	HookDirAfter  = "after"
)

type Hook struct {
	Prefix   HookPrefix
	Process  string
	Name     string
	F        ProcessFunc
	Override bool
	Target   string
	Dir      string
}

func NewHook(prefix HookPrefix, process string, name string, f ProcessFunc) Hook {
	return Hook{
		Prefix:  prefix,
		Process: process,
		Name:    name,
		F:       f,
	}
}

func (ho Hook) SetOverride(b bool) Hook {
	ho.Override = b

	return ho
}

func (ho Hook) SetDir(target, d string) Hook {
	ho.Target = target
	ho.Dir = d

	return ho
}

func (ho Hook) Add(ps *Processes) error {
	switch ho.Dir {
	case HookDirBefore, HookDirAfter:
		if len(ho.Target) < 1 {
			return xerrors.Errorf("target is empty for setting direction")
		} else if ho.Dir != HookDirBefore && ho.Dir != HookDirAfter {
			return xerrors.Errorf("unknown dir, %q", ho.Dir)
		}
	}

	switch ho.Dir {
	case HookDirBefore:
		return ps.AddHookBefore(ho.Prefix, ho.Process, ho.Name, ho.Target, ho.F, ho.Override)
	case HookDirAfter:
		return ps.AddHookAfter(ho.Prefix, ho.Process, ho.Name, ho.Target, ho.F, ho.Override)
	default:
		return ps.AddHook(ho.Prefix, ho.Process, ho.Name, ho.F, ho.Override)
	}
}

func EmptyHookFunc(ctx context.Context) (context.Context, error) {
	return ctx, nil
}
