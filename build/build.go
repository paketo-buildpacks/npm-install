package build

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	libbuildpackV3 "github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libjavabuildpack"
	"github.com/cloudfoundry/npm-cnb/detect"
	"github.com/cloudfoundry/npm-cnb/utils"
	"github.com/fatih/color"
)

func CreateLaunchMetadata() libbuildpackV3.LaunchMetadata {
	return libbuildpackV3.LaunchMetadata{
		Processes: libbuildpackV3.Processes{
			libbuildpackV3.Process{
				Type:    "web",
				Command: "npm start",
			},
		},
	}
}

type ModuleInstaller interface {
	InstallInCache(string, string) error
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
	bp, ok := builder.BuildPlan[detect.NPMDependency]
	if !ok {
		return Modules{}, false, nil
	}

	modules := Modules{
		npm:         npm,
		app:         builder.Application,
		logger:      builder.Logger,
		cacheLayer:  builder.Cache.Layer(detect.NPMDependency),
		launchLayer: builder.Launch.Layer(detect.NPMDependency),
	}

	if val, ok := bp.Metadata["build"]; ok {
		modules.buildContribution = val.(bool)
	}

	if val, ok := bp.Metadata["launch"]; ok {
		modules.launchContribution = val.(bool)
	}

	return modules, true, nil
}

func (m Modules) Contribute() error {
	if m.buildContribution {
		return fmt.Errorf("do not set build to true as part of the build plan when using the npm buildpack")
	}

	if !m.launchContribution {
		return nil
	}

	appModulesDir := filepath.Join(m.app.Root, "node_modules")
	cacheModulesDir := filepath.Join(m.cacheLayer.Root, "node_modules")
	launchModulesDir := filepath.Join(m.launchLayer.Root, "node_modules")

	vendored, err := libjavabuildpack.FileExists(appModulesDir)
	if err != nil {
		return fmt.Errorf("failed to check for vendored node_modules: %v", err)
	}

	sameSHASums, err := m.packageLockMatchesMetadataSha()
	if err != nil {
		return fmt.Errorf("failed in checking shas: %v", err)
	}

	boldNode := color.New(color.FgBlue, color.Bold).Sprint("Node Modules")
	if !sameSHASums {
		m.logger.FirstLine("%s: %s to launch", boldNode, color.YellowString("Contributing"))

		if vendored {
			m.logger.FirstLine("Removing cached node_modules")
			if err := os.RemoveAll(cacheModulesDir); err != nil {
				return fmt.Errorf("failed to remove cached node_modules: %v", err)
			}

			m.logger.FirstLine("%s: %s to cache", boldNode, color.YellowString("Copying"))
			if err := m.copyModulesToLayer(appModulesDir, cacheModulesDir); err != nil {
				return fmt.Errorf("failed to copy node_modules to the cache: %v", err)
			}

			m.logger.FirstLine("%s: %s", boldNode, color.YellowString("Rebuilding"))
			if err := m.npm.Rebuild(m.cacheLayer.Root); err != nil {
				return fmt.Errorf("failed to rebuild node_modules: %v", err)
			}
		} else {
			m.logger.FirstLine("%s: %s to cache", color.New(color.FgBlue, color.Bold).Sprint("package.json"), color.YellowString("Copying"))

			if err := os.MkdirAll(m.cacheLayer.Root, 0777); err != nil {
				return fmt.Errorf("failed to create directory %s : %v", m.cacheLayer.Root, err)
			}

			appPackageJsonPath := filepath.Join(m.app.Root, "package.json")
			cachePackageJsonPath := filepath.Join(m.cacheLayer.Root, "package.json")
			if err := utils.CopyFile(appPackageJsonPath, cachePackageJsonPath); err != nil {
				return fmt.Errorf("failed to copy package.json : %v", err)
			}

			appPackageLockPath := filepath.Join(m.app.Root, "package-lock.json")
			cachePackageLockPath := filepath.Join(m.cacheLayer.Root, "package-lock.json")
			if err := utils.CopyFile(appPackageLockPath, cachePackageLockPath); err != nil {
				return fmt.Errorf("failed to copy package-lock.json: %v", err)
			}

			m.logger.FirstLine("%s: %s", boldNode, color.YellowString("Installing"))
			if err := m.npm.InstallInCache(m.app.Root, m.cacheLayer.Root); err != nil {
				return fmt.Errorf("failed to install node_modules: %v", err)
			}
		}

		if err := m.copyModulesToLayer(cacheModulesDir, launchModulesDir); err != nil {
			return fmt.Errorf("failed to copy the node_modules to the launch layer: %v", err)
		}

		if err := m.writeMetadataSha(filepath.Join(m.app.Root, "package-lock.json")); err != nil {
			return fmt.Errorf("failed to write metadata to package-lock.json: %v", err)
		}
	} else {
		m.logger.FirstLine("%s: %s cached launch layer", boldNode, color.GreenString("Reusing"))
	}

	m.logger.SubsequentLine("Cleaning up node_modules")
	if err := os.RemoveAll(appModulesDir); err != nil {
		return fmt.Errorf("failed to clean up the node_modules: %v", err)
	}

	m.logger.SubsequentLine("Writing NODE_PATH for node_modules")
	if err := m.launchLayer.WriteProfile("NODE_PATH", fmt.Sprintf("export NODE_PATH=%s", launchModulesDir)); err != nil {
		return fmt.Errorf("failed to write NODE_PATH in the launch layer: %v", err)
	}

	return nil
}

func (m Modules) packageLockMatchesMetadataSha() (bool, error) {
	packageLockPath := filepath.Join(m.app.Root, "package-lock.json")
	if exists, err := libjavabuildpack.FileExists(packageLockPath); err != nil {
		return false, fmt.Errorf("failed to check for package-lock.json: %v", err)
	} else if !exists {
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

func (m Modules) writeMetadataSha(path string) error {
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

func (m *Modules) copyModulesToLayer(src, dest string) error {
	if exist, err := libjavabuildpack.FileExists(dest); err != nil {
		return err
	} else if !exist {
		if err := os.MkdirAll(dest, 0777); err != nil {
			return err
		}
	}

	if err := utils.CopyDirectory(src, dest); err != nil {
		return err
	}

	return nil
}
