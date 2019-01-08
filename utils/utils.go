package utils

import (
	"os"
	"os/exec"
)

type CommandRunner struct {
}

func (r CommandRunner) Run(bin, dir string, args ...string) error {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
