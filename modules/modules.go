package modules

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libcfbuildpack/helper"

	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/layers"
)

const Dependency = "modules"

type PackageManager interface {
	Install(cache, location string) error
	Rebuild(location string) error
}

type Metadata struct {
	Hash string
}

func (m Metadata) Identity() (name string, version string) {
	return Dependency, m.Hash
}

type Contributor struct {
	Metadata           Metadata
	buildContribution  bool
	launchContribution bool
	pkgManager         PackageManager
	app                application.Application
	layer              layers.Layer
	launch             layers.Layers
}

func NewContributor(context build.Build, pkgManager PackageManager) (Contributor, bool, error) {
	plan, shouldUseNPM := context.BuildPlan[Dependency]
	if !shouldUseNPM {
		return Contributor{}, false, nil
	}

	lockFile := filepath.Join(context.Application.Root, "package-lock.json")
	if exists, err := helper.FileExists(lockFile); err != nil {
		return Contributor{}, false, err
	} else if !exists {
		return Contributor{}, false, fmt.Errorf(`unable to find "package-lock.json"`)
	}

	buf, err := ioutil.ReadFile(lockFile)
	if err != nil {
		return Contributor{}, false, err
	}

	hash := sha256.Sum256(buf)

	contributor := Contributor{
		app:        context.Application,
		pkgManager: pkgManager,
		layer:      context.Layers.Layer(Dependency),
		launch:     context.Layers,
		Metadata:   Metadata{hex.EncodeToString(hash[:])},
	}

	if _, ok := plan.Metadata["build"]; ok {
		contributor.buildContribution = true
	}

	if _, ok := plan.Metadata["launch"]; ok {
		contributor.launchContribution = true
	}

	return contributor, true, nil
}

func (c Contributor) Contribute() error {
	return c.layer.Contribute(c.Metadata, func(layer layers.Layer) error {
		nodeModules := filepath.Join(c.app.Root, "node_modules")

		vendored, err := helper.FileExists(nodeModules)
		if err != nil {
			return fmt.Errorf("unable to stat node_modules: %s", err.Error())
		}

		if vendored {
			c.layer.Logger.Info("Rebuilding node_modules")
			if err := c.pkgManager.Rebuild(c.app.Root); err != nil {
				return fmt.Errorf("unable to rebuild node_modules: %s", err.Error())
			}
		} else {
			c.layer.Logger.Info("Installing node_modules")
			if err := c.pkgManager.Install(layer.Root, c.app.Root); err != nil {
				return fmt.Errorf("unable to install node_modules: %s", err.Error())
			}
		}

		if err := os.MkdirAll(layer.Root, 0777); err != nil {
			return fmt.Errorf("unable make layer: %s", err.Error())
		}

		if err := helper.CopyDirectory(nodeModules, filepath.Join(layer.Root, "node_modules")); err != nil {
			return fmt.Errorf(`unable to copy "%s" to "%s": %s`, nodeModules, layer.Root, err.Error())
		}

		if err := os.RemoveAll(nodeModules); err != nil {
			return fmt.Errorf("unable to remove node_modules from the app dir: %s", err.Error())
		}

		npmCache := filepath.Join(c.app.Root, "npm-cache")

		if exists, err := helper.FileExists(npmCache); err != nil {
			return err
		} else if exists {
			if err := helper.CopyDirectory(npmCache, filepath.Join(layer.Root, "npm-cache")); err != nil {
				return fmt.Errorf(`unable to copy "%s" to "%s": %s`, npmCache, layer.Root, err.Error())
			}

			if err := os.RemoveAll(npmCache); err != nil {
				return fmt.Errorf("unable to remove existing npm-cache: %s", err.Error())
			}
		}

		if err := layer.OverrideSharedEnv("NODE_PATH", filepath.Join(layer.Root, "node_modules")); err != nil {
			return err
		}

		return c.launch.WriteMetadata(layers.Metadata{
			Processes: []layers.Process{{"web", "npm start"}},
		})
	}, c.flags()...)
}

func (c Contributor) flags() []layers.Flag {
	flags := []layers.Flag{layers.Cache}

	if c.buildContribution {
		flags = append(flags, layers.Build)
	}

	if c.launchContribution {
		flags = append(flags, layers.Launch)
	}

	return flags
}
