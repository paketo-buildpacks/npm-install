package modules

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/layers"
)

const Dependency = "modules"

type PackageManager interface {
	Install(location string) error
	Rebuild(location string) error
}

type Contributor struct {
	buildContribution  bool
	launchContribution bool
	pkgManager         PackageManager
	app                application.Application
	layer              layers.Layer
	launch             layers.Layers
	id                 string
}

func NewContributor(builder build.Build, pkgManager PackageManager) (Contributor, bool, error) {
	plan, shouldUseNPM := builder.BuildPlan[Dependency]
	if !shouldUseNPM {
		return Contributor{}, false, nil
	}

	lockFile := filepath.Join(builder.Application.Root, "package-lock.json")
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
		app:        builder.Application,
		pkgManager: pkgManager,
		layer:      builder.Layers.Layer(Dependency),
		launch:     builder.Layers,
		id:         hex.EncodeToString(hash[:]),
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
	return c.layer.Contribute(c, func(layer layers.Layer) error {
		nodeModules := filepath.Join(c.app.Root, "node_modules")

		vendored, err := helper.FileExists(nodeModules)
		if err != nil {
			return fmt.Errorf("unable to stat node_modules: %s", err.Error())
		}

		if vendored {
			if err := c.pkgManager.Rebuild(c.app.Root); err != nil {
				return fmt.Errorf("unable to rebuild node_modules: %s", err.Error())
			}
		} else {
			if err := c.pkgManager.Install(c.app.Root); err != nil {
				return fmt.Errorf("unable to install node_modules: %s", err.Error())
			}
		}

		if err := os.MkdirAll(layer.Root, 0777); err != nil {
			return fmt.Errorf("unable make layer: %s", err.Error())
		}

		if err := helper.CopyDirectory(nodeModules, layer.Root); err != nil {
			return fmt.Errorf(`unable to copy "%s" to "%s": %s`, nodeModules, layer.Root, err.Error())
		}

		if err := os.RemoveAll(nodeModules); err != nil {
			return fmt.Errorf("unable to remove node_modules from the app dir: %s", err.Error())
		}

		if err := layer.OverrideSharedEnv("NODE_PATH", layer.Root); err != nil {
			return err
		}

		return c.launch.WriteMetadata(layers.Metadata{
			Processes: []layers.Process{{"web", "npm start"}},
		})
	}, c.flags()...)
}

func (c Contributor) Identity() (name string, version string) {
	return Dependency, c.id
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
