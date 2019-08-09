package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"syscall"

	"github.com/inconshreveable/log15"
	"github.com/spf13/cobra"

	"github.com/spikeekips/mitum/common"
	contest_module "github.com/spikeekips/mitum/contrib/contest/module"
	"github.com/spikeekips/mitum/isaac"
	"github.com/spikeekips/mitum/keypair"
	"github.com/spikeekips/mitum/network"
	"github.com/spikeekips/mitum/node"
	"github.com/spikeekips/mitum/seal"
)

var (
	sigc           chan os.Signal
	memProfileFile *os.File
)

var rootCmd = &cobra.Command{
	Use:   "contest",
	Short: "contest is the consensus tester of ISAAC+",
	Args:  cobra.NoArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// set logging
		var logOutput string
		var handler log15.Handler
		if len(FlagLogOut) > 0 {
			// check FlagLogOut is directory
			fi, err := os.Stat(FlagLogOut)
			if err != nil {
				cmd.Println("Error:", err.Error())
				os.Exit(1)
			}

			var logOutput string = FlagLogOut
			switch mode := fi.Mode(); {
			case mode.IsDir():
				//
			case mode.IsRegular():
				logOutput = filepath.Base(FlagLogOut)
			}

			handler = LogFileByNodeHandler(
				filepath.Clean(logOutput),
				common.LogFormatter(flagLogFormat.f),
				flagQuiet,
			)
		} else {
			handler, _ = common.LogHandler(common.LogFormatter(flagLogFormat.f), FlagLogOut)
		}
		handler = log15.CallerFileHandler(handler)
		handler = log15.LvlFilterHandler(flagLogLevel.lvl, handler)

		logs := []log15.Logger{
			log,
			common.Log(),
			isaac.Log(),
			keypair.Log(),
			network.Log(),
			node.Log(),
			seal.Log(),
			contest_module.Log(),
		}
		for _, l := range logs {
			common.SetLogger(l, flagLogLevel.lvl, handler)
		}

		log.Debug("parsed flags", "flags", printFlags(cmd, flagLogFormat.f))

		if len(logOutput) > 0 {
			log.Debug("output log", "directory", logOutput)
		}

		if len(flagCPUProfile) > 0 {
			f, err := os.Create(flagCPUProfile)
			if err != nil {
				panic(err)
			}
			if err := pprof.StartCPUProfile(f); err != nil {
				panic(err)
			}
			log.Debug("cpuprofile enabled")
		}

		if len(flagMemProfile) > 0 {
			f, err := os.Create(flagMemProfile)
			if err != nil {
				panic(err)
			}
			if err := pprof.WriteHeapProfile(f); err != nil {
				panic(err)
			}
			memProfileFile = f
			log.Debug("memprofile enabled")
		}

		sigc = make(chan os.Signal, 1)
		signal.Notify(
			sigc,
			syscall.SIGTERM,
			syscall.SIGQUIT,
		)

		go func() {
			s := <-sigc
			if len(flagCPUProfile) > 0 {
				pprof.StopCPUProfile()
				log.Debug("cpuprofile closed")
			}

			if len(flagMemProfile) > 0 {
				memProfileFile.Close()
				log.Debug("cpuprofile closed")
			}

			log.Info("contest stopped by force", "sig", s)
			os.Exit(0)
		}()
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if len(flagCPUProfile) > 0 {
			pprof.StopCPUProfile()
			log.Debug("cpuprofile closed")
		}
	},
}

func main() {
	rootCmd.PersistentFlags().Var(&flagLogLevel, "log-level", "log level: {debug error warn info crit}")
	rootCmd.PersistentFlags().Var(&flagLogFormat, "log-format", "log format: {json terminal}")
	rootCmd.PersistentFlags().StringVar(&FlagLogOut, "log", FlagLogOut, "log output directory")
	rootCmd.PersistentFlags().StringVar(&flagCPUProfile, "cpuprofile", flagCPUProfile, "write cpu profile to file")
	rootCmd.PersistentFlags().StringVar(&flagMemProfile, "memprofile", flagMemProfile, "write memory profile to file")
	rootCmd.PersistentFlags().BoolVar(&flagQuiet, "quiet", flagQuiet, "quiet")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(0)
}
