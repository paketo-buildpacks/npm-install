package modules

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/buildpack/libbuildpack/application"
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
)

type PackageManager interface {
	CI(cacheLayer, moduleLayer, location string) error
	Install(cacheLayer, moduleLayer, location string) error
	Rebuild(cacheLayer, location string) error
	WarnUnmetDependencies(appRoot string) error
}

type Contributor struct {
	buildContribution  bool
	launchContribution bool
	pkgManager         PackageManager
	app                application.Application
	nodeModulesLayer   layers.Layer
	npmCacheLayer      layers.Layer
	launch             layers.Layers
}

func NewContributor(context build.Build, pkgManager PackageManager) (Contributor, bool, error) {
	plan, wantDependency, err := context.Plans.GetShallowMerged(Dependency)
	if err != nil {
		return Contributor{}, false, err
	}

	if !wantDependency {
		return Contributor{}, false, nil
	}

	contributor := Contributor{
		app:              context.Application,
		pkgManager:       pkgManager,
		nodeModulesLayer: context.Layers.Layer(Dependency),
		npmCacheLayer:    context.Layers.Layer(Cache),
		launch:           context.Layers,
	}

	contributor.buildContribution, _ = plan.Metadata["build"].(bool)
	contributor.launchContribution, _ = plan.Metadata["launch"].(bool)

	return contributor, true, nil
}

func (c Contributor) Contribute(now time.Time) error {
	sum := NewTimeChecksum(now)

	file, err := os.Open(filepath.Join(c.app.Root, PackageLock))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer file.Close()

	if file != nil {
		sum = NewChecksum(file)
	}

	hash, err := sum.String()
	if err != nil {
		return err
	}

	if err := c.nodeModulesLayer.Contribute(NewMetadata(ModulesMetaName, hash), c.contributeNodeModules, c.flags()...); err != nil {
		return err
	}

	if err := c.npmCacheLayer.Contribute(NewMetadata(CacheMetaName, hash), c.contributeNPMCache, layers.Cache); err != nil {
		return err
	}

	return c.launch.WriteApplicationMetadata(layers.Metadata{
		Processes: []layers.Process{
			{
				Type:    "web",
				Command: "npm start",
				Direct:  false,
			},
		},
	})
}

func (c Contributor) contributeNodeModules(layer layers.Layer) error {
	nodeModules := filepath.Join(c.app.Root, ModulesDir)

	if err := c.tipVendorDependencies(nodeModules); err != nil {
		return err
	}

	locked, err := helper.FileExists(filepath.Join(c.app.Root, PackageLock))
	if err != nil {
		return fmt.Errorf("unable to stat node_modules: %s", err.Error())
	}

	cached, err := helper.FileExists(filepath.Join(c.app.Root, CacheDir))
	if err != nil {
		return fmt.Errorf("unable to stat node_modules: %s", err.Error())
	}

	vendored, err := helper.FileExists(nodeModules)
	if err != nil {
		return fmt.Errorf("unable to stat node_modules: %s", err.Error())
	}

	switch {
	case !locked && vendored, locked && vendored && !cached:
		c.nodeModulesLayer.Logger.Info("running npm rebuild")
		if err := c.pkgManager.Rebuild(c.npmCacheLayer.Root, c.app.Root); err != nil {
			return fmt.Errorf("unable to rebuild node_modules: %s", err.Error())
		}

	case !locked && !vendored:
		c.nodeModulesLayer.Logger.Info("running npm install")
		if err := c.pkgManager.Install(layer.Root, c.npmCacheLayer.Root, c.app.Root); err != nil {
			return fmt.Errorf("unable to install node_modules: %s", err.Error())
		}

	case locked:
		c.nodeModulesLayer.Logger.Info("running npm ci")
		if err := c.pkgManager.CI(layer.Root, c.npmCacheLayer.Root, c.app.Root); err != nil {
			return fmt.Errorf("unable to install node_modules: %s", err.Error())
		}
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

	if err := layer.OverrideSharedEnv("NPM_CONFIG_PRODUCTION", "true"); err != nil {
		return err
	}

	if err := layer.OverrideSharedEnv("NPM_CONFIG_LOGLEVEL", "error"); err != nil {
		return err
	}

	return layer.AppendPathSharedEnv("PATH", filepath.Join(layer.Root, ModulesDir, ".bin"))
}

func (c *Contributor) tipVendorDependencies(nodeModules string) error {
	subdirs, err := hasSubdirs(nodeModules)
	if err != nil {
		return err
	}
	if !subdirs {
		c.nodeModulesLayer.Logger.Info("It is recommended to vendor the application's Node.js dependencies")
	}

	return nil
}

func hasSubdirs(path string) (bool, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	for _, file := range files {
		if file.IsDir() {
			return true, nil
		}
	}

	return false, nil
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
