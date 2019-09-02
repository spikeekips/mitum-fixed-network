package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"golang.org/x/xerrors"
)

var (
	flagLogLevel      FlagLogLevel  = FlagLogLevel{lvl: zerolog.DebugLevel}
	flagLogFormat     FlagLogFormat = FlagLogFormat{f: "json"}
	FlagLogOut        string
	flagCPUProfile    string
	flagMemProfile    string
	flagTrace         string
	flagExitAfter     time.Duration
	flagNumberOfNodes uint = 3
	flagQuiet         bool
	flagQueries       []string
	flagJSONPretty    bool
)

type FlagLogLevel struct {
	lvl zerolog.Level
}

func (f FlagLogLevel) String() string {
	return f.lvl.String()
}

func (f *FlagLogLevel) Set(v string) error {
	lvl, err := zerolog.ParseLevel(v)
	if err != nil {
		return err
	}

	f.lvl = lvl

	return nil
}

func (f FlagLogLevel) Type() string {
	return "log-level"
}

type FlagLogFormat struct {
	f string
}

func (f FlagLogFormat) String() string {
	return f.f
}

func (f *FlagLogFormat) Set(v string) error {
	s := strings.ToLower(v)
	switch s {
	case "json":
	case "terminal":
	default:
		return xerrors.Errorf("invalid log format: %q", v)
	}

	f.f = s

	return nil
}

func (f FlagLogFormat) Type() string {
	return "log-format"
}

func printError(cmd *cobra.Command, err error) {
	fmt.Fprintf(os.Stderr, "error: %s\n\n", err.Error())
	_ = cmd.Help()
}

func escapeFlagValue(v interface{}, q string) string {
	if len(q) < 1 {
		return fmt.Sprintf("%v", v)
	}

	return q +
		strings.Replace(fmt.Sprintf("%v", v), "'", "\\"+q, -1) + q
}

func printFlagsJSON(cmd *cobra.Command) json.RawMessage {
	out := map[string]interface{}{}

	cmd.Flags().VisitAll(func(pf *pflag.Flag) {
		if pf.Name == "help" {
			return
		}

		out[fmt.Sprintf("--%s", pf.Name)] = map[string]interface{}{
			"default": escapeFlagValue(pf.DefValue, ""),
			"value":   escapeFlagValue(pf.Value, ""),
		}
	})

	b, _ := json.Marshal(out)

	return b
}
