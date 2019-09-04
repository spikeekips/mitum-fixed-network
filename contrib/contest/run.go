package main

import (
	"os"

	"github.com/spf13/cobra"
)

func run(cmd *cobra.Command, nodes *Nodes) error {
	defer func() {
		if err := nodes.Stop(); err != nil {
			cmd.Println("Error: failed to stop nodes:", err.Error())
			os.Exit(1)
		}
	}()

	if err := nodes.Start(); err != nil {
		return err
	}

	select {}

	return nil
}
