package main

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "contest config",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config, err := LoadConfig(args[0])
		if err != nil {
			cmd.Println("Error:", err.Error())
			os.Exit(1)
		}

		b, err := yaml.Marshal(config)
		fmt.Println(">> err", err)
		fmt.Println(">> config", string(b))
	},
}

var configFullCmd = &cobra.Command{
	Use:   "full",
	Short: "full config",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		config, err := LoadConfig(args[0])
		if err != nil {
			cmd.Println("Error:", err.Error())
			os.Exit(1)
		}

		b, err := yaml.Marshal(config)
		fmt.Println("-x" + strings.Repeat("-", int(TermWidth())-2))
		fmt.Println(string(bytes.TrimSpace(b)))
		fmt.Println(strings.Repeat("-", int(TermWidth())-2) + "x-")
	},
}

var configCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "check config",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		_, err := LoadConfig(args[0])
		if err != nil {
			cmd.Println("Error:", err.Error())
			os.Exit(1)
		}
	},
}

func init() {
	configCmd.AddCommand(configCheckCmd)
	configCmd.AddCommand(configFullCmd)
	rootCmd.AddCommand(configCmd)
}
