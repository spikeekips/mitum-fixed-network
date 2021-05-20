package pm

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/util"
)

func (pm *Processes) Processed(pr string) bool {
	_, found := pm.processed[pr]

	return found
}

func emptyProcessFunc(ctx context.Context) (context.Context, error) {
	return ctx, nil
}

func MustBaseProcess(name string, requires []string, f ProcessFunc) BaseProcess {
	pr, err := NewProcess(name, requires, f)
	if err != nil {
		panic(err)
	}

	return pr
}

func MustEmptyProcess(name string, requires []string) BaseProcess {
	return MustBaseProcess(name, requires, emptyProcessFunc)
}

type testProcesses struct {
	suite.Suite
}

func (t *testProcesses) TestCirculation() {
	cases := []struct {
		processes []BaseProcess
		err       string
	}{
		{
			processes: []BaseProcess{
				MustEmptyProcess("a", []string{"b", "c"}),
			},
			err: "not found",
		},
		{
			processes: []BaseProcess{
				MustEmptyProcess("a", []string{"b", "c"}),
				MustEmptyProcess("b", nil),
				MustEmptyProcess("c", nil),
			},
		},
		{
			processes: []BaseProcess{
				MustEmptyProcess("a", []string{"b", "c"}),
				MustEmptyProcess("c", []string{"b"}),
				MustEmptyProcess("b", nil),
			},
		},
		{
			processes: []BaseProcess{
				MustEmptyProcess("a", []string{"b", "c"}),
				MustEmptyProcess("b", nil),
				MustEmptyProcess("c", []string{"a"}),
			},
			err: `circulation found: "a"`,
		},
		{
			processes: []BaseProcess{
				MustEmptyProcess("a", []string{"b", "c"}),
				MustEmptyProcess("b", []string{"a"}),
				MustEmptyProcess("c", nil),
			},
			err: `circulation found: "a"`,
		},
		{
			processes: []BaseProcess{
				MustEmptyProcess("a", []string{"b", "c"}),
				MustEmptyProcess("b", []string{"c"}),
				MustEmptyProcess("c", []string{"b"}),
			},
			err: `circulation found: "b"`,
		},
		{
			processes: []BaseProcess{
				MustEmptyProcess("a", []string{"b", "c"}),
				MustEmptyProcess("b", nil),
				MustEmptyProcess("c", []string{"b"}),
			},
		},
	}

	for i, c := range cases {
		i := i
		c := c
		if !t.Run(
			fmt.Sprintf("%d", i),
			func() {
				pm := NewProcesses().SetContext(context.Background())
				for _, pr := range c.processes {
					err := pm.AddProcess(pr, true)
					t.NoError(err)
				}

				err := pm.check()
				if err != nil {
					if len(c.err) > 0 {
						t.Contains(err.Error(), c.err, "%d", i)
					} else {
						t.NoError(err, "%d", i)
					}

					return
				} else if len(c.err) > 0 {
					t.NoError(xerrors.Errorf("expected error, but not occurred"), "%d; expected error=%q", i, c.err)

					return
				}
			},
		) {
			break
		}
	}
}

type testProcess struct {
	i        string
	rs       []string
	err      string
	disabled bool
}

func newTestProcess(i string, err string, rs ...string) testProcess {
	return testProcess{i: i, err: err, rs: rs}
}

func (t testProcess) Name() string {
	return t.i
}

func (t testProcess) Requires() []string {
	return t.rs
}

func (t testProcess) Disabled() bool {
	return t.disabled
}

func (t testProcess) SetDisabled(b bool) testProcess {
	t.disabled = b

	return t
}

func (t testProcess) IsValid([]byte) error {
	return nil
}

func (t testProcess) Run(ctx context.Context) (context.Context, error) {
	if len(t.err) > 0 {
		return nil, xerrors.Errorf(t.err)
	}

	var result []string
	if r := ctx.Value("r"); r != nil {
		result = r.([]string)
	}

	result = append(result, string(t.i))

	//lint:ignore SA1029 test
	return context.WithValue(ctx, "r", result), nil
}

type testProcessHook struct {
	prefix  HookPrefix
	process string
	value   string
}

func newtestProcessHook(prefix HookPrefix, process string, value string) testProcessHook {
	return testProcessHook{prefix: prefix, process: process, value: value}
}

func (tp testProcessHook) F() ProcessFunc {
	return func(ctx context.Context) (context.Context, error) {
		var result []string
		if r := ctx.Value("r"); r != nil {
			result = r.([]string)
		}

		result = append(result, fmt.Sprintf("%v", tp.value))

		//lint:ignore SA1029 test
		return context.WithValue(ctx, "r", result), nil
	}
}

func (t *testProcesses) TestRunSequence() {
	cases := []struct {
		processes []testProcess
		err       string
		context   []string
	}{
		{
			processes: []testProcess{
				newTestProcess("a", ""),
				newTestProcess("b", "", "a"),
				newTestProcess("c", "", "a"),
			},
			context: []string{"a", "b", "c"},
		},
		{
			processes: []testProcess{
				newTestProcess("a", ""),
				newTestProcess("b", "", "a"),
				newTestProcess("c", ""),
			},
			context: []string{"a", "b", "c"},
		},
		{
			processes: []testProcess{
				newTestProcess("b", "", "a"),
				newTestProcess("a", ""),
				newTestProcess("c", ""),
			},
			context: []string{"a", "b", "c"},
		},
		{
			processes: []testProcess{
				newTestProcess("b", ""),
				newTestProcess("a", "", "c"),
				newTestProcess("c", ""),
			},
			context: []string{"b", "c", "a"},
		},
		{
			processes: []testProcess{
				newTestProcess("a", ""),
				newTestProcess("b", "show me", "a"),
				newTestProcess("c", "", "a"),
			},
			err:     "show me",
			context: []string{"a"},
		},
		{
			processes: []testProcess{
				newTestProcess("a", ""),
				newTestProcess("b", "", "a"),
				newTestProcess("c", "show me", "a").SetDisabled(true), // will not be run
			},
			context: []string{"a", "b"},
		},
		{
			processes: []testProcess{
				newTestProcess("a", "").SetDisabled(true),
				newTestProcess("b", "", "a"),
				newTestProcess("c", "", "a"),
			},
			err: `process, "b" requires "a", but disabled`,
		},
	}

	for i, c := range cases {
		i := i
		c := c
		if !t.Run(
			fmt.Sprintf("%d", i),
			func() {
				pm := NewProcesses().SetContext(context.Background())
				for _, pr := range c.processes {
					err := pm.AddProcess(pr, true)
					t.NoError(err)
				}

				err := pm.Run()
				if err != nil {
					if len(c.err) > 0 {
						t.Contains(err.Error(), c.err, "%d", i)
					} else {
						t.NoError(err, "%d", i)
					}
				} else if len(c.err) > 0 {
					t.NoError(xerrors.Errorf("expected error, but not occurred"), "%d; expected error=%q", i, c.err)
				}

				if c.context != nil {
					if pm.Context() == nil {
						t.NoError(xerrors.Errorf("empty result context"))
					} else {
						result := pm.Context().Value("r").([]string)

						t.Equal(c.context, result, "%d; expected=%v result=%v", i, c.context, result)
					}
				}
			},
		) {
			break
		}
	}
}

func (t *testProcesses) TestInit() {
	pm := NewProcesses().SetContext(context.Background())

	processes := []testProcess{
		newTestProcess("a", ""),
		newTestProcess("b", "", "init", "a"),
		newTestProcess("c", "", "a"),
	}

	for _, pr := range processes {
		t.NoError(pm.AddProcess(pr, false))
	}

	{ // without init
		t.NoError(pm.Run())
		result := pm.Context().Value("r").([]string)

		t.Equal([]string{"a", "b", "c"}, result)
		t.True(pm.Processed(INITProcess))
	}

	{ // with init
		pm.SetINIT(func(ctx context.Context) (context.Context, error) {
			//lint:ignore SA1029 test
			ctx = context.WithValue(ctx, "r", []string{"killme"})

			return ctx, nil
		})
		t.NoError(pm.Run())
		result := pm.Context().Value("r").([]string)

		t.True(pm.Processed(INITProcess))

		t.Equal([]string{"killme", "a", "b", "c"}, result)
	}

	{ // Run() initializes context
		pm.SetINIT(func(ctx context.Context) (context.Context, error) {
			return ctx, nil
		})
		t.NoError(pm.Run())

		result := pm.Context().Value("r").([]string)

		t.Equal([]string{"a", "b", "c"}, result)
		t.True(pm.Processed(INITProcess))
	}
}

func (t *testProcesses) TestNilContext() {
	pm := NewProcesses().SetContext(context.Background())

	t.NoError(pm.AddProcess(newTestProcess("a", ""), false))

	pm.SetINIT(func(ctx context.Context) (context.Context, error) {
		return nil, nil
	})
	t.NoError(pm.Run())

	result := pm.Context().Value("r").([]string)
	t.Equal([]string{"a"}, result)
}

func (t *testProcesses) TestHooksSequence() {
	cases := []struct {
		processes []testProcess
		hooks     []testProcessHook
		err       string
		context   []string
	}{
		{
			processes: []testProcess{
				newTestProcess("a", ""),
				newTestProcess("b", "", "a"),
				newTestProcess("c", "", "a"),
			},
			hooks: []testProcessHook{
				newtestProcessHook(HookPrefixPre, INITProcess, "pre-init"),
				newtestProcessHook(HookPrefixPre, "a", "pre-a"),
				newtestProcessHook(HookPrefixPost, "a", "post-a"),
				newtestProcessHook(HookPrefixPost, "c", "post-c"),
			},
			context: []string{"pre-init", "pre-a", "a", "post-a", "b", "c", "post-c"},
		},
		{
			processes: []testProcess{
				newTestProcess("a", ""),
				newTestProcess("b", "", "a"),
				newTestProcess("c", "", "a"),
			},
			hooks: []testProcessHook{
				newtestProcessHook(HookPrefixPre, "a", "pre-a"),
				newtestProcessHook(HookPrefixPost, "a", "post-a"),
				newtestProcessHook(HookPrefixPost, "c", "post-c"),
			},
			context: []string{"pre-a", "a", "post-a", "b", "c", "post-c"},
		},
		{
			processes: []testProcess{
				newTestProcess("a", ""),
				newTestProcess("b", "", "c", "a"),
				newTestProcess("c", "", "a"),
			},
			hooks: []testProcessHook{
				newtestProcessHook(HookPrefixPre, "a", "pre-a"),
				newtestProcessHook(HookPrefixPost, "a", "post-a"),
				newtestProcessHook(HookPrefixPost, "c", "post-c"),
			},
			context: []string{"pre-a", "a", "post-a", "c", "post-c", "b"},
		},
	}

	for i, c := range cases {
		i := i
		c := c
		if !t.Run(
			fmt.Sprintf("%d", i),
			func() {
				pm := NewProcesses().SetContext(context.Background())
				for _, pr := range c.processes {
					err := pm.AddProcess(pr, true)
					t.NoError(err)
				}

				for _, ev := range c.hooks {
					t.NoError(pm.AddHook(ev.prefix, ev.process, util.UUID().String(), ev.F(), false))
				}

				err := pm.Run()
				if err != nil {
					if len(c.err) > 0 {
						t.Contains(err.Error(), c.err, "%d", i)
					} else {
						t.NoError(err, "%d", i)
					}
				} else if len(c.err) > 0 {
					t.NoError(xerrors.Errorf("expected error, but not occurred"), "%d; expected error=%q", i, c.err)
				}

				if c.context != nil {
					result := pm.Context().Value("r").([]string)

					t.Equal(c.context, result, "%d; expected=%v result=%v", i, c.context, result)
				}
			},
		) {
			break
		}
	}
}

func (t *testProcesses) TestAddBefore() {
	pm := NewProcesses().SetContext(context.Background())

	pr := newTestProcess("a", "")
	err := pm.AddProcess(pr, true)
	t.NoError(err)

	{
		ev := newtestProcessHook(HookPrefixPre, "a", "pre-a0")
		t.NoError(pm.AddHook(ev.prefix, ev.process, util.UUID().String(), ev.F(), false))
	}

	ev0 := newtestProcessHook(HookPrefixPre, "a", "pre-a")
	target := "target"
	t.NoError(pm.AddHook(ev0.prefix, ev0.process, target, ev0.F(), false))

	{
		ev := newtestProcessHook(HookPrefixPre, "a", "pre-a1")
		t.NoError(pm.AddHook(ev.prefix, ev.process, util.UUID().String(), ev.F(), false))
	}

	ev1 := newtestProcessHook(HookPrefixPre, "a", "pre-before-a")
	h0 := "new-hook"
	t.NoError(pm.AddHookBefore(ev1.prefix, ev1.process, h0, target, ev1.F(), false))

	t.NoError(pm.Run())

	result := pm.Context().Value("r").([]string)
	t.Equal([]string{"pre-a0", "pre-before-a", "pre-a", "pre-a1", "a"}, result)
}

func (t *testProcesses) TestAddAfter() {
	pm := NewProcesses().SetContext(context.Background())

	pr := newTestProcess("a", "")
	err := pm.AddProcess(pr, true)
	t.NoError(err)

	{
		ev := newtestProcessHook(HookPrefixPre, "a", "pre-a0")
		t.NoError(pm.AddHook(ev.prefix, ev.process, util.UUID().String(), ev.F(), false))
	}

	ev0 := newtestProcessHook(HookPrefixPre, "a", "pre-a")
	target := "target"
	t.NoError(pm.AddHook(ev0.prefix, ev0.process, target, ev0.F(), false))

	{
		ev := newtestProcessHook(HookPrefixPre, "a", "pre-a1")
		t.NoError(pm.AddHook(ev.prefix, ev.process, util.UUID().String(), ev.F(), false))
	}

	ev1 := newtestProcessHook(HookPrefixPre, "a", "pre-after-a")
	h0 := "new-hook"
	t.NoError(pm.AddHookAfter(ev1.prefix, ev1.process, h0, target, ev1.F(), false))

	t.NoError(pm.Run())

	result := pm.Context().Value("r").([]string)
	t.Equal([]string{"pre-a0", "pre-a", "pre-after-a", "pre-a1", "a"}, result)
}

func TestProcesses(t *testing.T) {
	suite.Run(t, new(testProcesses))
}
