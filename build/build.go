package build

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
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

type Metadata struct {
	SHA256 string `toml:"sha256"`
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
		modules.cacheLayer = builder.Cache.Layer(detect.ModulesDependency)
	}

	if val, ok := bp.Metadata["launch"]; ok {
		modules.launchContribution = val.(bool)
		modules.launchLayer = builder.Launch.Layer(detect.ModulesDependency)
	}

	return modules, true, nil
}

func (m Modules) PackageLockMatchesMetadataSha() (bool, error) {
	packageLockPath := filepath.Join(m.app.Root, "package-lock.json")
	exists, err := libjavabuildpack.FileExists(packageLockPath)
	if err != nil {
		return false, fmt.Errorf("failed to check for package-lock.json: %v", err)
	}

	if !exists {
		return false, fmt.Errorf("there is no package-lock.json in the app")
	}

	packageLockSha := sha256.New()
	if buf, err := ioutil.ReadFile(packageLockPath); err != nil {
		return false, fmt.Errorf("failed to read metadata: %v", err)
	} else {
		packageLockSha.Write(buf)
	}

	var metadata Metadata
	m.launchLayer.ReadMetadata(&metadata)
	metadataHash, err := hex.DecodeString(metadata.SHA256)
	if err != nil {
		return false, err
	}

	return bytes.Equal(metadataHash, packageLockSha.Sum(nil)), nil
}

func (m Modules) WriteMetadataSha(path string) error {
	sha := sha256.New()
	if buf, err := ioutil.ReadFile(path); err != nil {
		return fmt.Errorf("failed to read %s: %v", path, err)
	} else {
		if _, err := sha.Write(buf); err != nil {
			return err
		}
	}

	return m.launchLayer.WriteMetadata(Metadata{
		SHA256: hex.EncodeToString(sha.Sum(nil)),
	})
}

func (m Modules) Contribute() error {
	if !m.launchContribution {
		return nil
	}

	appModulesDir := filepath.Join(m.app.Root, "node_modules")
	vendored, err := libjavabuildpack.FileExists(appModulesDir)
	if err != nil {
		return fmt.Errorf("failed to check for vendored node_modules: %v", err)
	}

	sameShas, err := m.PackageLockMatchesMetadataSha()
	if err != nil {
		return fmt.Errorf("failed in checking shas: %v", err)
	}

	cacheDir := filepath.Join(m.cacheLayer.Root, "node_modules")
	launchDir := filepath.Join(m.launchLayer.Root, "node_modules")

	if !sameShas {
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

		if m.buildContribution {
			m.logger.FirstLine("%s: %s to cache",
				color.New(color.FgBlue, color.Bold).Sprint("Node Modules"), color.YellowString("Contributing"))
			if err := m.copyModulesToLayer(cacheDir); err != nil {
				return fmt.Errorf("failed to copy node_modules to the cache layer: %v", err)
			}
		}

		if m.launchContribution {
			m.logger.FirstLine("%s: %s to launch",
				color.New(color.FgBlue, color.Bold).Sprint("Node Modules"), color.YellowString("Contributing"))
			if err := m.copyModulesToLayer(launchDir); err != nil {
				return fmt.Errorf("failed to copy the node_modules to the launch layer: %v", err)
			}
		}

		if err := m.WriteMetadataSha(filepath.Join(m.app.Root, "package-lock.json")); err != nil {
			return fmt.Errorf("failed to write metadata to package-lock.json: %v", err)
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
