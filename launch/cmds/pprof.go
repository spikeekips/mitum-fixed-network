package cmds

import (
	"os"
	"runtime/pprof"
	"runtime/trace"

	"github.com/alecthomas/kong"
	"golang.org/x/xerrors"
)

var PprofVars = kong.Vars{
	"enable_pprof":     "false",
	"mem_pprof_file":   "mitum-mem.pprof",
	"cpu_pprof_file":   "mitum-cpu.pprof",
	"trace_pprof_file": "mitum-trace.pprof",
}

type PprofFlags struct {
	EnableProfiling bool   `name:"enable-pprof" help:"enable profiling (default:${enable_pprof})" default:"${enable_pprof}"`       // nolint
	MemProf         string `name:"pprof-mem" help:"memory prof file (default:${mem_pprof_file})" default:"${mem_pprof_file}"`      // nolint
	CPUProf         string `name:"pprof-cpu" help:"cpu prof file (default:${cpu_pprof_file})" default:"${cpu_pprof_file}"`         // nolint
	TraceProf       string `name:"pprof-trace" help:"trace prof file (default:${trace_pprof_file})" default:"${trace_pprof_file}"` // nolint
}

func RunPprofs(flags *PprofFlags) (func() error, error) {
	if !flags.EnableProfiling {
		return func() error {
			return nil
		}, nil
	}

	var exitHooks []func() error
	if len(flags.TraceProf) > 0 {
		if c, err := RunTracePprof(flags.TraceProf); err != nil {
			return nil, err
		} else {
			exitHooks = append(exitHooks, c)
		}
	}

	if len(flags.CPUProf) > 0 {
		if c, err := RunCPUPprof(flags.CPUProf); err != nil {
			return nil, err
		} else {
			exitHooks = append(exitHooks, c)
		}
	}

	if len(flags.MemProf) > 0 {
		if c, err := RunMemPprof(flags.MemProf); err != nil {
			return nil, err
		} else {
			exitHooks = append(exitHooks, c)
		}
	}

	return func() error {
		var errs []string
		for i := range exitHooks {
			if err := exitHooks[i](); err != nil {
				errs = append(errs, err.Error())
			}
		}
		if len(errs) > 0 {
			return xerrors.Errorf("failed to close profiling: %v", errs)
		}

		return nil
	}, nil
}

func RunTracePprof(s string) (func() error, error) {
	if f, err := os.Create(s); err != nil {
		return nil, err
	} else if err := trace.Start(f); err != nil {
		return nil, err
	} else {
		return func() error {
			trace.Stop()

			if err := f.Close(); err != nil {
				return xerrors.Errorf("failed to close trace prof file, %s: %w", s, err)
			}

			return nil
		}, nil
	}
}

func RunCPUPprof(s string) (func() error, error) {
	if f, err := os.Create(s); err != nil {
		return nil, err
	} else if err := pprof.StartCPUProfile(f); err != nil {
		return nil, err
	} else {
		return func() error {
			pprof.StopCPUProfile()

			if err := f.Close(); err != nil {
				return xerrors.Errorf("failed to close cpu prof file, %s: %w", s, err)
			}

			return nil
		}, nil
	}
}

func RunMemPprof(s string) (func() error, error) {
	if f, err := os.Create(s); err != nil {
		return nil, err
	} else if err := pprof.WriteHeapProfile(f); err != nil {
		return nil, err
	} else {
		return func() error {
			if err := f.Close(); err != nil {
				return xerrors.Errorf("failed to close mem prof file, %s: %w", s, err)
			}

			return nil
		}, nil
	}
}
