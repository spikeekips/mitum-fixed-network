package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/spf13/cobra"

	"github.com/spikeekips/mitum/contrib/contest/condition"
)

var queryCmd = &cobra.Command{
	Use:   "query <log>",
	Short: "query logs",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		lf, err := os.Open(args[0])
		if err != nil {
			cmd.Println("Error: failed to open log file:", err.Error())
			os.Exit(1)
		}
		defer lf.Close() // nolint

		log.Debug().
			Strs("query", flagQueries).
			Msg("query")

		var conditionChecker *condition.MultipleConditionChecker
		if len(flagQueries) > 0 {
			cc, err := condition.NewMultipleConditionChecker(flagQueries, 1)
			if err != nil {
				cmd.Println("Error: wrong query:", err.Error())
				os.Exit(1)
			}
			conditionChecker = cc
		}

		satisfiedChan := make(chan bool, 1)
		lw := condition.NewLogWatcher(conditionChecker, satisfiedChan)
		_ = lw.Start()

		var wait sync.WaitGroup
		wait.Add(1)
		go func() {
			reader := bufio.NewReader(lf)

		end:
			for {
				select {
				case <-satisfiedChan:
					wait.Done()
					break end
				default:
					b, err := reader.ReadBytes('\n')
					if err != nil {
						wait.Done()
						break end
					}
					lw.Write(b)
				}
			}
		}()
		wait.Wait()

		hw := condition.NewHighlightWriter(os.Stdout)

		var enc *json.Encoder
		if flagJSONPretty {
			enc = json.NewEncoder(hw)
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "  ")
		}

		for _, o := range conditionChecker.Satisfied() {
			for _, li := range o {
				if enc != nil {
					enc.Encode(json.RawMessage(li.Bytes()))
				} else {
					fmt.Fprint(hw, string(li.Bytes()))
				}
			}
		}
	},
}

func init() {
	queryCmd.Flags().StringArrayVar(&flagQueries, "query", nil, "query")
	queryCmd.Flags().BoolVar(&flagJSONPretty, "pretty", false, "pretty json output")

	rootCmd.AddCommand(queryCmd)
}
