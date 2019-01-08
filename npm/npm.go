package npm

import (
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/helper"
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

func (n NPM) Install(cache, location string) error {
	nodeModules, existingNodeModules := filepath.Join(location, "node_modules"), filepath.Join(cache, "node_modules")
	if exists, err := helper.FileExists(existingNodeModules); err != nil {
		return err
	} else if exists {
		n.Logger.Info("Reusing existing node_modules")
		if err := helper.CopyDirectory(existingNodeModules, nodeModules); err != nil {
			return err
		}
		defer os.RemoveAll(existingNodeModules)
	}

	npmCache, existingNPMCache := filepath.Join(location, "npm-cache"), filepath.Join(cache, "npm-cache")
	if exists, err := helper.FileExists(existingNPMCache); err != nil {
		return err
	} else if exists {
		n.Logger.Info("Reusing existing npm-cache")
		if err := helper.CopyDirectory(existingNPMCache, npmCache); err != nil {
			return err
		}
		defer os.RemoveAll(existingNPMCache)
	}

	if err := n.Runner.Run("npm", location, "install", "--unsafe-perm", "--cache", npmCache); err != nil {
		return err
	}

	return n.Runner.Run("npm", location, "cache", "verify", "--cache", npmCache)
}

func (n NPM) Rebuild(location string) error {
	return n.Runner.Run("npm", location, "rebuild")
}
