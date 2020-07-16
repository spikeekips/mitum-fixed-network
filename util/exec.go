package util

import (
	"context"
	"os/exec"
)

func ShellExec(ctx context.Context, c string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "bash", "-c", c) // nolint: gosec

	return cmd.CombinedOutput()
}
