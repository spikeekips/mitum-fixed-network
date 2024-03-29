package pm

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/spikeekips/mitum/util/logging"
)

type HookPrefix string

const (
	HookPrefixPre  HookPrefix = "pre"
	HookPrefixPost HookPrefix = "post"
	INITProcess               = "init"
)

type Processes struct {
	*logging.Logging
	ctx            context.Context
	ctxSource      context.Context
	processesOrder []string
	processes      map[string]Process
	hooksByProcess map[string] /* process */ []string /* hooks */
	hooks          map[string] /* hook */ ProcessFunc
	processed      map[string]struct{}
}

func NewProcesses() *Processes {
	pm := &Processes{
		Logging: logging.NewLogging(func(c zerolog.Context) zerolog.Context {
			return c.Str("module", "process-manager")
		}),
		processes:      map[string]Process{},
		ctxSource:      context.Background(),
		hooksByProcess: map[string][]string{},
		hooks:          map[string]ProcessFunc{},
	}

	return pm.SetINIT(
		func(ctx context.Context) (context.Context, error) { return ctx, nil },
	)
}

func (pm *Processes) Context() context.Context {
	return pm.ctx
}

func (pm *Processes) ContextSource() context.Context {
	return pm.ctxSource
}

func (pm *Processes) SetContext(ctx context.Context) *Processes {
	pm.ctxSource = ctx

	return pm
}

func (pm *Processes) SetINIT(f ProcessFunc) *Processes {
	pr, _ := NewProcess(INITProcess, nil, f)

	pm.processes[pr.Name()] = pr

	return pm
}

func (pm *Processes) AddProcess(pr Process, override bool) error {
	if pr.Name() == INITProcess {
		return errors.Errorf("use SetINIT")
	}

	if _, found := pm.processes[pr.Name()]; found {
		if !override {
			return errors.Errorf("process already added, %q", pr.Name())
		}
	} else {
		pm.processesOrder = append(pm.processesOrder, pr.Name())
	}

	pm.processes[pr.Name()] = pr

	return nil
}

func (pm *Processes) RemoveProcess(name string) error {
	if name == INITProcess {
		return errors.Errorf("can not remove init process")
	}

	if _, found := pm.processes[name]; !found {
		return errors.Errorf("process not found, %q", name)
	}

	processesOrder := make([]string, len(pm.processesOrder)-1)
	var c int
	for i := range pm.processesOrder {
		if pm.processesOrder[i] == name {
			continue
		}
		processesOrder[c] = pm.processesOrder[i]
		c++
	}

	pm.processesOrder = processesOrder

	delete(pm.processes, name)

	return nil
}

func (pm *Processes) AddHook(
	prefix HookPrefix,
	pr string,
	hook string,
	f ProcessFunc,
	override bool,
) error {
	prName := pm.processHookName(prefix, pr)
	if _, found := pm.hooks[hook]; found {
		if !override {
			return errors.Errorf("hook already added, %q", hook)
		}
	} else {
		pm.hooksByProcess[prName] = append(pm.hooksByProcess[prName], hook)
	}

	pm.hooks[hook] = f

	return nil
}

func (pm *Processes) AddHookBefore(
	prefix HookPrefix,
	pr, hook, target string,
	f ProcessFunc,
	override bool,
) error {
	if err := pm.AddHook(prefix, pr, hook, f, override); err != nil {
		return err
	}

	prName := pm.processHookName(prefix, pr)
	b := make([]string, len(pm.hooksByProcess[prName]))

	var i int
	for _, k := range pm.hooksByProcess[prName] {
		if k == target {
			b[i] = hook
			i++
		} else if k == hook {
			continue
		}

		b[i] = k
		i++
	}

	pm.hooksByProcess[prName] = b

	return nil
}

func (pm *Processes) AddHookAfter(
	prefix HookPrefix,
	pr, hook, target string,
	f ProcessFunc,
	override bool,
) error {
	if err := pm.AddHook(prefix, pr, hook, f, override); err != nil {
		return err
	}

	prName := pm.processHookName(prefix, pr)
	b := make([]string, len(pm.hooksByProcess[prName]))

	var i int
	for _, k := range pm.hooksByProcess[prName] {
		if k == hook {
			continue
		}

		b[i] = k
		i++

		if k == target {
			b[i] = hook
			i++
		}
	}

	pm.hooksByProcess[prName] = b

	return nil
}

func (pm *Processes) Run() error {
	if err := pm.check(); err != nil {
		return err
	}

	pm.processed = map[string]struct{}{}
	pm.ctx = pm.ctxSource

	// run init first
	if err := pm.runProcess(INITProcess, ""); err != nil {
		return errors.Wrap(err, "failed to run init")
	}

	for _, name := range pm.processesOrder {
		if err := pm.runProcess(name, INITProcess); err != nil {
			return errors.Wrapf(err, "failed to run process, %q", name)
		}
	}

	return nil
}

func (pm *Processes) check() error {
	// NOTE check circulation of requires
	processed := map[string]struct{}{}
	requireCount := map[string]int{}

	for _, name := range pm.processesOrder {
		if pr, found := pm.processes[name]; !found {
			return errors.Errorf("process, %q not found", name)
		} else if err := pm.checkProcess(pr, processed, requireCount); err != nil {
			return errors.Wrapf(err, "failed to check process, %q", name)
		}
	}

	return nil
}

func (pm *Processes) checkProcess(pr Process, processed map[string]struct{}, requireCount map[string]int) error {
	if _, found := processed[pr.Name()]; found {
		return nil
	} else if pr.Disabled() {
		return nil
	}

	if c := requireCount[pr.Name()]; c > 0 {
		return errors.Errorf("circulation found: %q", pr.Name())
	}
	requireCount[pr.Name()] = 1

	for _, r := range pr.Requires() {
		if npr, found := pm.processes[r]; !found {
			return errors.Errorf("process, %q requires %q, but not found", pr.Name(), r)
		} else if npr.Disabled() {
			return errors.Errorf("process, %q requires %q, but disabled", pr.Name(), r)
		} else if err := pm.checkProcess(npr, processed, requireCount); err != nil {
			return err
		}
	}

	processed[pr.Name()] = struct{}{}

	return nil
}

func (pm *Processes) runProcess(s, from string) error {
	if _, found := pm.processed[s]; found {
		return nil
	}

	var pr Process
	switch i, found := pm.processes[s]; {
	case !found:
		return errors.Errorf("process, %q not found", s)
	case i.Disabled():
		return nil
	default:
		pr = i
	}

	l := pm.Log().With().Str("process", pr.Name()).Str("from_process", from).Logger()
	l.Debug().Msg("trying to run")

	// run requires
	for _, r := range pr.Requires() {
		if err := pm.runProcess(r, pr.Name()); err != nil {
			return err
		}
	}

	// check pre hooks
	if err := pm.runProcessHooks(HookPrefixPre, pr.Name()); err != nil {
		return err
	}

	ctx, err := pr.Run(pm.ctx)
	if err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}

	pm.ctx = ctx
	pm.processed[pr.Name()] = struct{}{}

	// check post hooks
	if err := pm.runProcessHooks(HookPrefixPost, pr.Name()); err != nil {
		return err
	}

	l.Debug().Msg("processed")

	return nil
}

func (pm *Processes) runProcessHooks(prefix HookPrefix, pr string) error {
	prHook := pm.processHookName(prefix, pr)

	var hooks []string
	switch i, found := pm.hooksByProcess[prHook]; {
	case !found:
		return nil
	case len(i) < 1:
		return nil
	default:
		hooks = i
	}

	for i := range hooks {
		h := hooks[i]
		pm.Log().Debug().Str("process", pr).Str("hook", h).Str("prefix", string(prefix)).Msg("trying to run hooks")
		if err := pm.runProcessHook(h, pr); err != nil {
			return err
		}
	}

	return nil
}

func (pm *Processes) runProcessHook(hook, from string) error {
	l := pm.Log().With().Str("hook", hook).Str("from", from).Logger()

	switch f, found := pm.hooks[hook]; {
	case !found:
		return errors.Errorf("hook, %q not found", hook)
	case f == nil:
		return nil
	default:
		ctx, err := f(pm.ctx)
		if err != nil {
			return errors.Wrapf(err, "failed to emit hook of %q(%s)", hook, from)
		}
		if ctx == nil {
			ctx = context.Background()
		}

		pm.ctx = ctx
	}

	l.Debug().Msg("hook processed")

	return nil
}

func (*Processes) processHookName(prefix HookPrefix, pr string) string {
	return fmt.Sprintf("%s:%s", prefix, pr)
}
