package util

import "os/exec"

func Exec(c string) error {
	cmd := exec.Command("bash", "-c", c) // nolint: gosec

	return cmd.Run()
}
