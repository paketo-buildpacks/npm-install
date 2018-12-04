package npm

import (
	"os"
	"os/exec"
	"path/filepath"
)

type NPM struct {
}

func (n NPM) Install(location string) error {
	return run(location, "install", "--unsafe-perm", "--cache", filepath.Join(location, "npm-cache"))
}

func (n NPM) Rebuild(location string) error {
	return run(location, "rebuild")
}

func run(dir string, args ...string) error {
	cmd := exec.Command("npm", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
