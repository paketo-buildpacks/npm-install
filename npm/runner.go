package npm

import (
	"os"
	"os/exec"
)

type Runner interface {
	Run(dir string, args ...string) error
}

type npmCmd struct{}

func (r *npmCmd) Run(dir string, args ...string) error {
	cmd := exec.Command("npm", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
