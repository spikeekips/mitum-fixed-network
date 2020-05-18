package cmds

import (
	"fmt"
	"os"
)

var Version string = "v0.1-proto3"

type VersionCommand struct {
}

func (cmd *VersionCommand) Run() error {
	_, _ = fmt.Fprintln(os.Stdout, Version)

	return nil
}
