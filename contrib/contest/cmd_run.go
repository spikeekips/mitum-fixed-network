package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/spikeekips/mitum/common"
	"github.com/spikeekips/mitum/contrib/contest/condition"
	"github.com/spikeekips/mitum/contrib/contest/configs"
	"github.com/spikeekips/mitum/node"
)

var runCmd = &cobra.Command{
	Use:   "run <config>",
	Short: "run contest",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("number-of-nodes") {
			if flagNumberOfNodes < 1 {
				cmd.Println("Error: `--number-of-nodes` should be greater than zero")
				os.Exit(1)
			}
		}

		log.Info().Msg("contest started")
		defer func() {
			log.Info().Msg("contest stopped")
		}()

		config, err := configs.LoadConfigFromFile(args[0], flagNumberOfNodes)
		if err != nil {
			cmd.Println("Error:", err.Error())
			os.Exit(1)
		} else if err = config.IsValid(); err != nil {
			cmd.Println("Error:", err.Error())
			os.Exit(1)
		} else if err = config.Merge(nil); err != nil {
			cmd.Println("Error:", err.Error())
			os.Exit(1)
		}

		log.Debug().
			Object("config", config).
			Dur("flagExitAfter", flagExitAfter).
			Msg("config loaded")

		var nodes *Nodes
		nodeList := getAllNodesFromConfig(config)

		previousExitHooks := exitHooks
		exitHooks = nil
		exitHooks = append(exitHooks, func() {
			_ = nodes.Stop()
		})

		if len(config.Conditions.Conditions) > 0 {
			satisfiedChan := make(chan bool)

			go func() {
				<-satisfiedChan
				sigc <- syscall.SIGINT
			}()

			checkers := prepareConditions(config, nodeList)
			cp := condition.NewMultipleConditionCheckers(checkers, 1)
			lw := condition.NewLogWatcher(cp, satisfiedChan)

			exitHooks = append(
				exitHooks,
				func() {
					_ = lw.Stop()

					satisfied := cp.AllSatisfied()
					log.Info().
						Bool("satisfied", satisfied).
						Msg("all satisfied?")

					if satisfied {
						printSatisfied(cp)
					} else {
						exitCode = 1
					}
				},
			)

			log = log.
				Output(io.MultiWriter(logOutput, lw))

			_ = lw.SetLogger(stdoutLog)
			_ = lw.Start()
		}
		exitHooks = append(exitHooks, previousExitHooks...)

		nodes, err = NewNodes(config, nodeList)
		if err != nil {
			printError(cmd, err)
			os.Exit(1)
		}

		go func() { // exit-after
			if flagExitAfter < time.Nanosecond {
				return
			}

			<-time.After(flagExitAfter)
			log.Info().
				Dur("expire", flagExitAfter).
				Msg("expired")
			sigc <- syscall.SIGINT // interrupt process by force after timeout
		}()

		if err := run(cmd, nodes); err != nil {
			printError(cmd, err)
			os.Exit(1)
		}
	},
}

func init() {
	runCmd.Flags().DurationVar(&flagExitAfter, "exit-after", 0, "exit after; 0 forever")
	runCmd.Flags().UintVar(&flagNumberOfNodes, "number-of-nodes", 0, "number of nodes")

	rootCmd.AddCommand(runCmd)
}

func prepareConditions(config *configs.Config, nodeList []node.Node) []condition.ConditionChecker {
	var checkers []condition.ConditionChecker

	all, found := config.Conditions.Conditions["all"]
	if found {
		for _, n := range nodeList {
			for _, ck := range all {
				nc := ck.Condition().Prepend(
					"and",
					condition.NewComparison("node", "=", []interface{}{n.Alias()}, reflect.String),
				)
				checkers = append(checkers, condition.NewConditionCheckerFromCondition(nc))
			}
		}
	}

	for k, cks := range config.Conditions.Conditions {
		if k == "all" {
			continue
		}

		checkers = append(checkers, cks...)
	}

	return checkers
}

func printSatisfied(cp *condition.MultipleConditionChecker) {
	allSatisfied := cp.Satisfied()

	color.NoColor = false
	hw := condition.NewHighlightWriter(os.Stdout)
	var enc *json.Encoder
	if flagJSONPretty {
		enc = json.NewEncoder(hw)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
	}

	termWidth := common.TermWidth()
	if termWidth < 1 {
		termWidth = 80
	}

	for q, os := range allSatisfied {
		fmt.Printf("%s %s\n", color.New(color.FgGreen).Sprintf("query:"), q)

		fmt.Println(color.New(color.FgGreen).Sprintf("matched log:"))
		for _, li := range os {
			if enc != nil {
				if err := enc.Encode(json.RawMessage(li.Bytes())); err != nil {
					log.Error().Err(err).Msg("failed to encode log item")
				}
			} else {
				fmt.Fprint(hw, string(li.Bytes()))
			}
		}
		fmt.Println(strings.Repeat("=", termWidth))
	}
}
