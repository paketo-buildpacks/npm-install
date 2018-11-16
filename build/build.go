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
	InstallToLayer(string, string) error
	RebuildLayer(string, string) error
	CleanAndCopyToDst(string, string) error
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
	if !m.buildContribution && !m.launchContribution {
		return nil
	}

	if m.buildContribution {
		if !m.launchContribution {
			m.logger.FirstLine("%s: %s to cache", logHeader(), color.YellowString("Contributing"))
			if err := m.installInCache(); err != nil {
				return fmt.Errorf("failed to install in cache for build : %v", err)
			}
		}

		m.logger.SubsequentLine("Writing NODE_PATH")
		if err := m.cacheLayer.AppendPathEnv("NODE_PATH", filepath.Join(m.cacheLayer.Root, "node_modules")); err != nil {
			return err
		}
	}

	if m.launchContribution {
		if sameSHASums, err := m.packageLockMatchesMetadataSha(); err != nil {
			return err
		} else if sameSHASums {
			m.logger.FirstLine("%s: %s cached launch layer", logHeader(), color.GreenString("Reusing"))
			return nil
		}

		m.logger.FirstLine("%s: %s to launch", logHeader(), color.YellowString("Contributing"))

		if err := m.installInCache(); err != nil {
			return fmt.Errorf("failed to install in cache for launch : %v", err)
		}

		if err := m.installInLaunch(); err != nil {
			return fmt.Errorf("failed to install in launch : %v", err)
		}

		if err := m.writeProfile(); err != nil {
			return fmt.Errorf("failed to write profile.d : %v", err)
		}
	}

	appModulesDir := filepath.Join(m.app.Root, "node_modules")
	if err := os.RemoveAll(appModulesDir); err != nil {
		return fmt.Errorf("failed to clean up the node_modules: %v", err)
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

	buf, err := ioutil.ReadFile(packageLockPath)
	if err != nil {
		return false, fmt.Errorf("failed to read package-lock.json: %v", err)
	}

	var metadata Metadata
	if err := m.launchLayer.ReadMetadata(&metadata); err != nil {
		return false, err
	}

	metadataHash, err := hex.DecodeString(metadata.SHA256)
	if err != nil {
		return false, err
	}

	hash := sha256.Sum256(buf)
	return bytes.Equal(metadataHash, hash[:]), nil
}

func (m Modules) writeMetadataSha(path string) error {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", path, err)
	}

	hash := sha256.Sum256(buf)
	return m.launchLayer.WriteMetadata(Metadata{SHA256: hex.EncodeToString(hash[:])})
}

func (m *Modules) copyModulesToLayer(src, dest string) error {
	if exist, err := libjavabuildpack.FileExists(dest); err != nil {
		return err
	} else if !exist {
		if err := os.MkdirAll(dest, 0777); err != nil {
			return err
		}
	}
	return utils.CopyDirectory(src, dest)
}

func (m Modules) installInCache() error {
	appModulesDir := filepath.Join(m.app.Root, "node_modules")

	vendored, err := libjavabuildpack.FileExists(appModulesDir)
	if err != nil {
		return fmt.Errorf("could not locate app modules directory : %s", err)
	}

	if vendored {
		m.logger.SubsequentLine("%s node_modules", color.YellowString("Rebuilding"))

		if err := m.npm.RebuildLayer(m.app.Root, m.cacheLayer.Root); err != nil {
			return fmt.Errorf("failed to rebuild node_modules: %v", err)
		}
	} else {
		m.logger.SubsequentLine("%s node_modules", color.YellowString("Installing"))

		cacheModulesDir := filepath.Join(m.cacheLayer.Root, "node_modules")
		if exists, err := libjavabuildpack.FileExists(cacheModulesDir); err != nil {
			return err
		} else if !exists {
			if err := os.MkdirAll(cacheModulesDir, 0777); err != nil {
				return fmt.Errorf("could not make node modules directory : %s", err)
			}
		}

		if err := os.Symlink(cacheModulesDir, appModulesDir); err != nil {
			return fmt.Errorf("could not symlink node modules directory : %s", err)
		}
		defer os.Remove(appModulesDir)

		if err := m.npm.InstallToLayer(m.app.Root, m.cacheLayer.Root); err != nil {
			return fmt.Errorf("failed to install and copy node_modules: %v", err)
		}
	}

	return nil
}

func (m Modules) installInLaunch() error {
	cacheModulesDir := filepath.Join(m.cacheLayer.Root, "node_modules")
	launchModulesDir := filepath.Join(m.launchLayer.Root, "node_modules")

	if err := m.npm.CleanAndCopyToDst(cacheModulesDir, launchModulesDir); err != nil {
		return fmt.Errorf("failed to copy the node_modules to the launch layer: %v", err)
	}

	if err := m.writeMetadataSha(filepath.Join(m.app.Root, "package-lock.json")); err != nil {
		return fmt.Errorf("failed to write metadata to package-lock.json: %v", err)
	}

	return nil
}

func (m Modules) writeProfile() error {
	m.logger.SubsequentLine("Writing profile.d/NODE_PATH")

	launchModulesDir := filepath.Join(m.launchLayer.Root, "node_modules")
	if err := m.launchLayer.WriteProfile("NODE_PATH", fmt.Sprintf("export NODE_PATH=%s", launchModulesDir)); err != nil {
		return fmt.Errorf("failed to write NODE_PATH in the launch layer: %v", err)
	}
	return nil
}

func logHeader() string {
	return color.New(color.FgBlue, color.Bold).Sprint("Node Modules")
}
