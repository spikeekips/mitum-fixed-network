package main

import (
	"os"

	"github.com/spf13/cobra"
)

func run(cmd *cobra.Command, config *Config) error {
	nodes, err := NewNodes(config)
	if err != nil {
		return err
	}

	defer func() {
		if err := nodes.Stop(); err != nil {
			cmd.Println("Error: failed to stop nodes: %s", err.Error())
			os.Exit(1)
		}
	}()

	if err := nodes.Start(); err != nil {
		return err
	}

	select {}

	return nil
}
