package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "contest config",
	Args:  cobra.NoArgs,
}

var configFullCmd = &cobra.Command{
	Use:   "full <config>",
	Short: "full config",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("number-of-nodes") {
			if flagNumberOfNodes < 1 {
				cmd.Println("Error: `--number-of-nodes` should be greater than zero")
				os.Exit(1)
			}
		}

		config, err := LoadConfig(args[0], flagNumberOfNodes)
		if err != nil {
			cmd.Println("Error:", err.Error())
			os.Exit(1)
		}

		b, err := yaml.Marshal(config)
		if err != nil {
			cmd.Println("Error:", err.Error())
			os.Exit(1)
		}

		fmt.Println("-x" + strings.Repeat("-", int(TermWidth())-2))
		fmt.Println(string(bytes.TrimSpace(b)))
		fmt.Println(strings.Repeat("-", int(TermWidth())-2) + "x-")
	},
}

var configCheckCmd = &cobra.Command{
	Use:   "check <config>",
	Short: "check config",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, err := LoadConfig(args[0], 0)
		if err != nil {
			cmd.Println("Error:", err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	configFullCmd.Flags().UintVar(&flagNumberOfNodes, "number-of-nodes", 0, "number of nodes")

	configCmd.AddCommand(configCheckCmd)
	configCmd.AddCommand(configFullCmd)
	rootCmd.AddCommand(configCmd)
}
