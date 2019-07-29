package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"syscall"
	"time"

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

var rootCmd = &cobra.Command{
	Use:   "contest",
	Short: "contest is the consensus tester of ISAAC+",
	Args:  cobra.NoArgs,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// set logging
		var logOutput string
		var handler log15.Handler
		if len(FlagLogOut) > 0 {
			logOutput = filepath.Join(FlagLogOut, common.Now().Format("20060102150405"))
			handler = LogFileByNodeHandler(
				logOutput,
				common.LogFormatter(flagLogFormat.f),
				flagQuiet,
			)

			latest := filepath.Join(FlagLogOut, "latest")
			if _, err := os.Lstat(latest); err == nil {
				if err := os.Remove(latest); err != nil {
					cmd.Println("Error:", err.Error())
					os.Exit(1)
				}
			}

			if err := os.Symlink(logOutput, latest); err != nil {
				cmd.Println("Error:", err.Error())
				os.Exit(1)
			}
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

		sigc := make(chan os.Signal, 1)
		signal.Notify(
			sigc,
			syscall.SIGTERM,
			syscall.SIGQUIT,
		)

		// exit-after
		go func() {
			if flagExitAfter < time.Nanosecond {
				return
			}

			<-time.After(flagExitAfter)
			sigc <- syscall.SIGINT // interrupt process by force after timeout
		}()

		go func() {
			s := <-sigc
			if len(flagCPUProfile) > 0 {
				pprof.StopCPUProfile()
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
	rootCmd.PersistentFlags().BoolVar(&flagQuiet, "quiet", flagQuiet, "quiet")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(0)
}
