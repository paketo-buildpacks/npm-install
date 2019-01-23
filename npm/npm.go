package npm

import (
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/npm-cnb/modules"
)

type Runner interface {
	Run(bin, dir string, args ...string) error
}

type Logger interface {
	Info(format string, args ...interface{})
}

type NPM struct {
	Runner Runner
	Logger Logger
}

func (n NPM) Install(modulesLayer, cacheLayer, location string) error {
	if err := n.moveDir(modulesLayer, location, modules.ModulesDir); err != nil {
		return err
	}

	if err := n.moveDir(cacheLayer, location, modules.CacheDir); err != nil {
		return err
	}

	npmCache := filepath.Join(location, modules.CacheDir)

	if err := n.Runner.Run("npm", location, "install", "--unsafe-perm", "--cache", npmCache); err != nil {
		return err
	}

	return n.Runner.Run("npm", location, "cache", "verify", "--cache", npmCache)
}

func (n NPM) Rebuild(location string) error {
	return n.Runner.Run("npm", location, "rebuild")
}

func (n NPM) moveDir(source, target, name string) error {
	dir := filepath.Join(source, name)
	if exists, err := helper.FileExists(dir); err != nil {
		return err
	} else if !exists {
		return nil
	}

	n.Logger.Info("Reusing existing %s", name)
	if err := helper.CopyDirectory(dir, filepath.Join(target, name)); err != nil {
		return err
	}

	return os.RemoveAll(dir)
}
