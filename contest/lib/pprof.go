package contestlib

import (
	"os"
	"runtime/pprof"
	"runtime/trace"

	"golang.org/x/xerrors"
)

type PprofFlags struct {
	MemProf   string `help:"memory prof file (default:${mem_prof_file})" default:"${mem_prof_file}"`
	CPUProf   string `help:"CPU prof file (default:${cpu_prof_file})" default:"${cpu_prof_file}"`
	TraceProf string `help:"trace prof file (default:${trace_prof_file})" default:"${trace_prof_file}"`
}

func RunPprof(flags *PprofFlags) (func() error, error) {
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
