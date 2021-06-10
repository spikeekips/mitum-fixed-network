package pm

import (
	"context"
	"fmt"

	"github.com/spikeekips/mitum/util/logging"
	"golang.org/x/xerrors"
)

type Hooks struct {
	*logging.Logging
	seq   []string
	hooks map[string] /* hook */ ProcessFunc
}

func NewHooks(name string) *Hooks {
	return &Hooks{
		Logging: logging.NewLogging(func(c logging.Context) logging.Emitter {
			return c.Str("module", fmt.Sprintf("hooks-%s", name))
		}),
		hooks: map[string]ProcessFunc{},
	}
}

func (hs *Hooks) Add(name string, f ProcessFunc, override bool) error {
	if _, found := hs.hooks[name]; found {
		if !override {
			return xerrors.Errorf("hook already added, %q", name)
		}
	}

	hs.hooks[name] = f
	hs.seq = append(hs.seq, name)

	return nil
}

func (hs *Hooks) AddBefore(name, target string, f ProcessFunc, override bool) error {
	if err := hs.Add(name, f, override); err != nil {
		return err
	}

	b := make([]string, len(hs.seq))

	var i int
	for _, k := range hs.seq {
		if k == target {
			b[i] = name
			i++
		} else if k == name {
			continue
		}

		b[i] = k
		i++
	}

	hs.seq = b

	return nil
}

func (hs *Hooks) AddAfter(name, target string, f ProcessFunc, override bool) error {
	if err := hs.Add(name, f, override); err != nil {
		return err
	}

	b := make([]string, len(hs.seq))

	var i int
	for _, k := range hs.seq {
		if k == name {
			continue
		}

		b[i] = k
		i++

		if k == target {
			b[i] = name
			i++
		}
	}

	hs.seq = b

	return nil
}

func (hs Hooks) Run(ctx context.Context) error {
	if len(hs.seq) < 1 {
		return nil
	}

	hs.Log().Debug().Msg("running hooks")

	for i := range hs.seq {
		name := hs.seq[i]
		i, err := hs.hooks[name](ctx)
		if err != nil {
			hs.Log().Error().Err(err).Str("hook", name).Msg("failed to run hook")

			return err
		}
		hs.Log().Debug().Str("hook", name).Msg("hook done")

		ctx = i
	}

	hs.Log().Debug().Msg("hooks done")

	return nil
}
