package npm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libcfbuildpack/buildpack"

	"github.com/Masterminds/semver"
	. "github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/npm-cnb/modules"
)

const UnmetDepWarning = "Unmet dependencies don't fail npm install but may cause runtime issues\nSee: https://github.com/npm/npm/issues/7494"

type Runner interface {
	Run(bin, dir string, quiet bool, args ...string) error
	RunWithOutput(bin, dir string, quiet bool, args ...string) (string, error)
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

	if err := n.runInstall(location, npmCache, false); err != nil {
		return err
	}
	return n.runCacheVerify(location, npmCache)
}

func (n NPM) Rebuild(cacheLayer, location string) error {
	if err := n.Runner.Run("npm", location, false, "rebuild"); err != nil {
		return fmt.Errorf("failed running npm rebuild %s", err.Error())
	}

	n.Logger.Info("Installing the additional un-vendored modules listed below:")
	npmCache := filepath.Join(location, modules.CacheDir)
	return n.runInstall(location, npmCache, true)
}

func (n NPM) moveDir(source, target, name string) error {
	dir := filepath.Join(source, name)
	if exists, err := FileExists(dir); err != nil {
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
	if err := CopyDirectory(dir, filepath.Join(target, name)); err != nil {
		return err
	}

	return os.RemoveAll(dir)
}

func (n NPM) getNPMVersion(location string) (buildpack.Version, error) {
	out, err := n.Runner.RunWithOutput("npm", location, true, "-v")
	if err != nil {
		return buildpack.Version{}, err
	}
	versionLimit, err := semver.NewVersion(out)
	if err != nil {
		return buildpack.Version{}, err
	}
	return buildpack.Version{Version: versionLimit}, nil
}

func (n NPM) runInstall(location string, cacheLocation string, skipAudit bool) error {
	args := []string{"install", "--unsafe-perm", "--cache", cacheLocation}
	if skipAudit {
		args = append(args, "--no-audit")
	}

	return n.Runner.Run("npm", location, false, args...)
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

	return n.Runner.Run("npm", location, false, "cache", "verify", "--cache", cacheLocation)
}

func (n NPM) WarnUnmetDependencies(appRoot string) error {
	output, err := n.Runner.RunWithOutput("npm", appRoot, true, "ls", "--depth=0")
	_, ok := err.(*exec.ExitError)
	if err != nil && !ok {
		return err
	}

	output = strings.ToLower(string(output))
	unmet := strings.Contains(output, "unmet dependency") || strings.Contains(output, "unmet peer dependency")
	if unmet {
		n.Logger.Info(UnmetDepWarning)
	}

	return nil
}
