package npm

import (
	"os"
	"os/exec"
)

type NPM struct{}

func (n *NPM) Install(src, dest string) error {
	return n.runCommand(src, "--prefix", dest, "install", "--unsafe-perm")
}

func (n *NPM) Rebuild(dir string) error {
	return n.runCommand(dir, "rebuild")
}

func (n *NPM) runCommand(dir string, args ...string) error {
	cmd := exec.Command("npm", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
