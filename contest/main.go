package main

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/diode"
	"golang.org/x/xerrors"

	"github.com/spikeekips/mitum/contest/commands"
	"github.com/spikeekips/mitum/contest/common"
)

var (
	log       zerolog.Logger
	exitHooks []func()
)

type Flags struct {
	Version struct {
	} `cmd:"" help:"Print version"`
	Run commands.RunCommand `cmd:"" help:"Run contest"`
	commands.CommonFlags
}

func printError(err error) {
	fmt.Fprintf(os.Stderr, "error: %+v\n", err)
}

func setupLogging(flags Flags) (zerolog.Logger, error) {
	zerolog.TimestampFieldName = "t"
	zerolog.TimeFieldFormat = time.RFC3339Nano
	zerolog.MessageFieldName = "m"

	zerolog.DisableSampling(true)

	var output io.Writer
	if flags.Log == nil || *flags.Log == "" {
		output = os.Stdout
	} else {
		f, err := os.OpenFile(filepath.Join(*flags.Log, "all.log"), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
		if err != nil {
			return zerolog.Logger{}, err
		}

		output = diode.NewWriter(
			f,
			1000,
			0,
			func(missed int) {
				fmt.Fprintf(os.Stderr, "zerolog: dropped %d log mesages", missed)
			},
		)

		exitHooks = append(exitHooks, func() {
			if l, ok := output.(diode.Writer); ok {
				_ = l.Close()
			}
		})
	}

	if flags.LogFormat == "terminal" {
		output = zerolog.ConsoleWriter{
			Out:        output,
			TimeFormat: time.RFC3339Nano,
		}
	}

	lc := zerolog.
		New(output).
		With().
		Timestamp()

	level := zerolog.Level(flags.LogLevel)
	if level == zerolog.DebugLevel {
		lc = lc.Caller().Stack()
	}

	return lc.Logger().Level(level), nil
}

func connectSignal() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(
		sigc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	go func() {
		s := <-sigc

		for _, h := range exitHooks {
			h()
		}

		log.Fatal().
			Str("sig", s.String()).
			Msg("contest stopped by force")

		os.Exit(1)
	}()

	select {}
}

func main() {
	flags := Flags{
		CommonFlags: commands.CommonFlags{},
	}

	ctx := kong.Parse(&flags,
		kong.Name("contest"),
		kong.Description("Consensus testing machine"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{
			NoAppSummary: false,
			Compact:      true,
			Summary:      false,
			Tree:         true,
		}),
		kong.Vars{
			"log":        "",
			"log_level":  "debug",
			"log_format": "terminal",
			"nodes":      "1",
		},
	)

	{
		l, err := setupLogging(flags)
		if err != nil {
			defer os.Exit(1)
			printError(err)
			return
		}
		log = l
	}

	log.Info().Msg("contest started")
	defer log.Info().Msg("contest stopped")

	log.Debug().Interface("flags", flags).Msg("flags parsed")

	if ctx.Command() == "version" {
		fmt.Fprintf(os.Stderr, "v0.0.1\n")
		return
	}

	err := ctx.Run(&flags.CommonFlags, &log, &exitHooks)
	if err != nil && xerrors.Is(err, common.LongRunningCommandError) {
		connectSignal()
	}

	ctx.FatalIfErrorf(err)
}
