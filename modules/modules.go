package modules

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
)

const (
	Dependency     = "node_modules"
	NodeDependency = "node"
	Cache          = "cache"
	ModulesDir     = "node_modules"
	CacheDir       = "npm-cache"
)

type PackageManager interface {
	Install(string, string, string) error
	Rebuild(string, string) error
	WarnUnmetDependencies(string) error
}

type MetadataInterface interface {
	Identity() (name string, version string)
}

type Metadata struct {
	Name string
	Hash string
}

func (m Metadata) Identity() (name string, version string) {
	return m.Name, m.Hash
}

type Contributor struct {
	NodeModulesMetadata MetadataInterface
	NPMCacheMetadata    MetadataInterface
	buildContribution   bool
	launchContribution  bool
	pkgManager          PackageManager
	app                 application.Application
	nodeModulesLayer    layers.Layer
	npmCacheLayer       layers.Layer
	launch              layers.Layers
}

func NewContributor(context build.Build, pkgManager PackageManager) (Contributor, bool, error) {
	plan, shouldUseNPM := context.BuildPlan[Dependency]
	if !shouldUseNPM {
		return Contributor{}, false, nil
	}

	contributor := Contributor{
		app:              context.Application,
		pkgManager:       pkgManager,
		nodeModulesLayer: context.Layers.Layer(Dependency),
		npmCacheLayer:    context.Layers.Layer(Cache),
		launch:           context.Layers,
	}

	lockFile := filepath.Join(context.Application.Root, "package-lock.json")
	if exists, err := helper.FileExists(lockFile); err != nil {
		return Contributor{}, false, err
	} else if exists {
		buf, err := ioutil.ReadFile(lockFile)
		if err != nil {
			return Contributor{}, false, err
		}
		hash := sha256.Sum256(buf)
		contributor.NodeModulesMetadata = Metadata{Dependency, hex.EncodeToString(hash[:])}
		contributor.NPMCacheMetadata = Metadata{Cache, hex.EncodeToString(hash[:])}
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
	if err := c.nodeModulesLayer.Contribute(c.NodeModulesMetadata, c.contributeNodeModules, c.flags()...); err != nil {
		return err
	}
	if err := c.npmCacheLayer.Contribute(c.NPMCacheMetadata, c.contributeNPMCache, layers.Cache); err != nil {
		return err
	}

	return c.launch.WriteApplicationMetadata(layers.Metadata{Processes: []layers.Process{{"web", "npm start"}}})
}

func (c Contributor) contributeNodeModules(layer layers.Layer) error {
	nodeModules := filepath.Join(c.app.Root, ModulesDir)

	vendored, err := helper.FileExists(nodeModules)
	if err != nil {
		return fmt.Errorf("unable to stat node_modules: %s", err.Error())
	}

	if vendored {
		c.nodeModulesLayer.Logger.Info("Rebuilding node_modules")
		if err := c.pkgManager.Rebuild(c.npmCacheLayer.Root, c.app.Root); err != nil {
			return fmt.Errorf("unable to rebuild node_modules: %s", err.Error())
		}
	} else {
		c.nodeModulesLayer.Logger.Info("Installing node_modules")
		if err := c.pkgManager.Install(layer.Root, c.npmCacheLayer.Root, c.app.Root); err != nil {
			return fmt.Errorf("unable to install node_modules: %s", err.Error())
		}
	}

	if err := os.MkdirAll(layer.Root, 0777); err != nil {
		return fmt.Errorf("unable make node modules layer: %s", err.Error())
	}

	nodeModulesExist, err := helper.FileExists(nodeModules)
	if err != nil {
		return fmt.Errorf("unable to stat node_modules: %s", err.Error())
	}

	if nodeModulesExist {
		if err := helper.CopyDirectory(nodeModules, filepath.Join(layer.Root, ModulesDir)); err != nil {
			return fmt.Errorf(`unable to copy "%s" to "%s": %s`, nodeModules, layer.Root, err.Error())
		}

		if err := os.RemoveAll(nodeModules); err != nil {
			return fmt.Errorf("unable to remove node_modules from the app dir: %s", err.Error())
		}
	}

	if err := os.Setenv("NODE_VERBOSE", "true"); err != nil {
		return fmt.Errorf("unable to set NODE_VERBOSE to true")
	}

	if err := c.pkgManager.WarnUnmetDependencies(c.app.Root); err != nil {
		return fmt.Errorf("failed to check unmet dependencies: %s", err.Error())
	}

	if err := layer.OverrideSharedEnv("NODE_PATH", filepath.Join(layer.Root, ModulesDir)); err != nil {
		return err
	}

	return layer.AppendPathSharedEnv("PATH", filepath.Join(layer.Root, ModulesDir, ".bin"))
}

func (c Contributor) contributeNPMCache(layer layers.Layer) error {
	if err := os.MkdirAll(layer.Root, 0777); err != nil {
		return fmt.Errorf("unable make npm cache layer: %s", err.Error())
	}

	npmCache := filepath.Join(c.app.Root, CacheDir)

	npmCacheExists, err := helper.FileExists(npmCache)
	if err != nil {
		return err
	}

	if npmCacheExists {
		if err := helper.CopyDirectory(npmCache, filepath.Join(layer.Root, CacheDir)); err != nil {
			return fmt.Errorf(`unable to copy "%s" to "%s": %s`, npmCache, layer.Root, err.Error())
		}

		if err := os.RemoveAll(npmCache); err != nil {
			return fmt.Errorf("unable to remove existing npm-cache: %s", err.Error())
		}
	}

	return nil
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
