package build

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"

	"github.com/cloudfoundry/npm-cnb/detect"

	"github.com/cloudfoundry/npm-cnb/utils"

	libbuildpackV3 "github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libjavabuildpack"
)

type ModuleInstaller interface {
	Install(string) error
	Rebuild(string) error
}

type Modules struct {
	buildContribution, launchContribution bool
	app                                   libbuildpackV3.Application
	cacheLayer                            libbuildpackV3.CacheLayer
	launchLayer                           libbuildpackV3.LaunchLayer
	logger                                libjavabuildpack.Logger
	npm                                   ModuleInstaller
}

func NewModules(builder libjavabuildpack.Build, npm ModuleInstaller) (Modules, bool, error) {
	bp, ok := builder.BuildPlan[detect.ModulesDependency]
	if !ok {
		return Modules{}, false, nil
	}

	modules := Modules{}
	modules.npm = npm

	modules.app = builder.Application
	modules.logger = builder.Logger

	if val, ok := bp.Metadata["build"]; ok {
		modules.buildContribution = val.(bool)
		modules.cacheLayer = builder.Cache.Layer("modules")
	}

	if val, ok := bp.Metadata["launch"]; ok {
		modules.launchContribution = val.(bool)
		modules.launchLayer = builder.Launch.Layer("modules")
	}

	return modules, true, nil
}

func (m Modules) Contribute() error {
	if !m.buildContribution && !m.launchContribution {
		return nil
	}

	appModulesDir := filepath.Join(m.app.Root, "node_modules")
	vendored, err := libjavabuildpack.FileExists(appModulesDir)
	if err != nil {
		return fmt.Errorf("failed to check for vendored node_modules: %v", err)
	}

	if vendored {
		m.logger.FirstLine("%s: %s",
			color.New(color.FgBlue, color.Bold).Sprint("Node Modules"), color.YellowString("Rebuilding"))
		if err := m.npm.Rebuild(m.app.Root); err != nil {
			return fmt.Errorf("failed to rebuild node_modules: %v", err)
		}
	} else {
		m.logger.FirstLine("%s: %s",
			color.New(color.FgBlue, color.Bold).Sprint("Node Modules"), color.YellowString("Installing"))
		if err := m.npm.Install(m.app.Root); err != nil {
			return fmt.Errorf("failed to install node_modules: %v", err)
		}
	}

	cacheDir := filepath.Join(m.cacheLayer.Root, "node_modules")
	if m.buildContribution {
		m.logger.FirstLine("%s: %s to cache",
			color.New(color.FgBlue, color.Bold).Sprint("Node Modules"), color.YellowString("Contributing"))
		if err := m.copyModulesToLayer(cacheDir); err != nil {
			return fmt.Errorf("failed to copy node_modules to the cache layer: %v", err)
		}
	}

	launchDir := filepath.Join(m.launchLayer.Root, "node_modules")
	if m.launchContribution {
		m.logger.FirstLine("%s: %s to launch",
			color.New(color.FgBlue, color.Bold).Sprint("Node Modules"), color.YellowString("Contributing"))
		if err := m.copyModulesToLayer(launchDir); err != nil {
			return fmt.Errorf("failed to copy the node_modules to the launch layer: %v", err)
		}
	}

	m.logger.SubsequentLine("Cleaning up node_modules")
	if err := os.RemoveAll(appModulesDir); err != nil {
		return fmt.Errorf("failed to clean up the node_modules: %v", err)
	}

	if m.launchContribution {
		m.logger.SubsequentLine("Creating symlink for node_modules")
		if err := os.Symlink(launchDir, appModulesDir); err != nil {
			return fmt.Errorf("failed to symlink the node_modules to the launch layer: %v", err)
		}
	}

	return nil
}

func (m *Modules) CreateLaunchMetadata() libbuildpackV3.LaunchMetadata {
	return libbuildpackV3.LaunchMetadata{
		Processes: libbuildpackV3.Processes{
			libbuildpackV3.Process{
				Type:    "web",
				Command: "npm start",
			},
		},
	}
}

func (m *Modules) copyModulesToLayer(dest string) error {
	if exist, err := libjavabuildpack.FileExists(dest); err != nil {
		return err
	} else if !exist {
		if err := os.MkdirAll(dest, 0777); err != nil {
			return err
		}
	}

	if err := utils.CopyDirectory(filepath.Join(m.app.Root, "node_modules"), dest); err != nil {
		return err
	}

	return nil
}
