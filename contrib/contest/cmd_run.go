package main

import (
	"os"
	"syscall"
	"time"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "run contest",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		log.Info("contest started")
		defer func() {
			log.Info("contest stopped")
		}()

		config, err := LoadConfig(args[0])
		if err != nil {
			cmd.Println("Error:", err.Error())
			os.Exit(1)
		}

		log.Debug("config loaded", "config", config)

		go func() { // exit-after
			if flagExitAfter < time.Nanosecond {
				return
			}

			<-time.After(flagExitAfter)
			sigc <- syscall.SIGINT // interrupt process by force after timeout
		}()

		if err := run(config); err != nil {
			printError(cmd, err)
		}
	},
}

func init() {
	runCmd.Flags().DurationVar(&flagExitAfter, "exit-after", 0, "exit after; 0 forever")
	runCmd.Flags().UintVar(&flagNumberOfNodes, "number-of-nodes", 0, "number of nodes")

	rootCmd.AddCommand(runCmd)
}
