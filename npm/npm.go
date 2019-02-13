package npm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/buildpack"

	"github.com/Masterminds/semver"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/npm-cnb/modules"
)

type Runner interface {
	Run(bin, dir string, args ...string) error
	RunWithOutput(bin, dir string, args ...string) (string, error)
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

	if err := n.runInstall(location, npmCache); err != nil {
		return err
	}
	return n.runCacheVerify(location, npmCache)
}

func (n NPM) Rebuild(cacheLayer, location string) error {
	if err := n.Runner.Run("npm", location, "rebuild"); err != nil {
		return fmt.Errorf("failed running npm rebuild %s", err.Error())
	}

	n.Logger.Info("Installing additional un-vendored modules")
	npmCache := filepath.Join(location, modules.CacheDir)
	return n.runInstall(location, npmCache)
}

func (n NPM) moveDir(source, target, name string) error {
	dir := filepath.Join(source, name)
	if exists, err := helper.FileExists(dir); err != nil {
		return err
	} else if !exists {
		return nil
	}

	var fullTarget, fullDir string
	var err error

	if fullTarget, err = filepath.Abs(filepath.Join(target, name)); err != nil {
		return err
	}
	if fullDir, err = filepath.Abs(dir); err != nil {
		return err
	}

	if fullDir == fullTarget {
		n.Logger.Info("Reusing identical %s", name)
		return nil
	}

	n.Logger.Info("Reusing existing %s", name)
	if err := helper.CopyDirectory(dir, filepath.Join(target, name)); err != nil {
		return err
	}

	return os.RemoveAll(dir)
}

func (n NPM) getNPMVersion(location string) (buildpack.Version, error) {
	out, err := n.Runner.RunWithOutput("npm", location, "-v")
	if err != nil {
		return buildpack.Version{}, err
	}
	versionLimit, err := semver.NewVersion(out)
	if err != nil {
		return buildpack.Version{}, err
	}
	return buildpack.Version{Version: versionLimit}, nil
}

func (n NPM) runInstall(location string, cacheLocation string) error {
	return n.Runner.Run("npm", location, "install", "--unsafe-perm", "--cache", cacheLocation)
}

func (n NPM) runCacheVerify(location, cacheLocation string) error {
	curVersion, err := n.getNPMVersion(location)
	if err != nil {
		return err
	}

	versionLimit, err := semver.NewVersion("5.0.0") //npm cache verify was added in npm 5.0.0
	if err != nil {
		return err
	}

	if curVersion.LessThan(versionLimit) {
		return nil
	}

	return n.Runner.Run("npm", location, "cache", "verify", "--cache", cacheLocation)
}
