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
	// WriteProfileD(string) error
	// WriteENV(string) error
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
		m.cacheLayer.AppendPathEnv("NODE_PATH", filepath.Join(m.cacheLayer.Root, "node_modules"))
		if !m.launchContribution {
			if err := m.installInCache(); err != nil {
				return fmt.Errorf("Failed to install in cache for build : %v", err)
			}
		}
	}

	if m.launchContribution {
		if err := m.writeProfile(); err != nil {
			return fmt.Errorf("Failed to write profile.d : %v", err)
		}

		sameSHASums, err := m.packageLockMatchesMetadataSha()
		if err != nil {
			return err
		}

		if sameSHASums {
			return nil
		}

		if err := m.installInCache(); err != nil {
			return fmt.Errorf("Failed to install in cache for launch : %v", err)
		}

		if err := m.installInLaunch(); err != nil {
			return fmt.Errorf("Failed to install in launch : %v", err)
		}
	}

	appModulesDir := filepath.Join(m.app.Root, "node_modules")
	m.logger.SubsequentLine("Removing node_modules from app")
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

func (m Modules) installInCache() error {
	appModulesDir := filepath.Join(m.app.Root, "node_modules")
	vendored, err := libjavabuildpack.FileExists(appModulesDir)
	if err != nil {
		return fmt.Errorf("could not locate app modules directory : %s", err)
	}

	if vendored {
		m.logger.SubsequentLine("%s cached node_modules", color.YellowString("Rebuilding"))
		if err := m.npm.RebuildLayer(m.app.Root, m.cacheLayer.Root); err != nil {
			return fmt.Errorf("failed to rebuild node_modules: %v", err)
		}
	} else {
		m.logger.SubsequentLine("%s node_modules", color.YellowString("Installing"))

		cacheModulesDir := filepath.Join(m.cacheLayer.Root, "node_modules")

		if exist, err := libjavabuildpack.FileExists(cacheModulesDir); err != nil {
			return err
		} else if !exist {
			if err := os.MkdirAll(cacheModulesDir, 0777); err != nil {
				return fmt.Errorf("could not make node modules directory : %s", err)
			}
		}

		if err := os.Symlink(cacheModulesDir, appModulesDir); err != nil {
			return fmt.Errorf("could not symlink node modules directory : %s", err)
		}
		defer func() error {
			if err := os.Remove(appModulesDir); err != nil {
				return err
			}
			return nil
		}()

		if err := m.npm.InstallToLayer(m.app.Root, m.cacheLayer.Root); err != nil {
			return fmt.Errorf("failed to install and copy node_modules: %v", err)
		}

	}

	return nil
}

func (m Modules) installInLaunch() error {
	sameSHASums, err := m.packageLockMatchesMetadataSha()
	if err != nil {
		return err
	}

	if sameSHASums {
		m.logger.SubsequentLine("app and launch layers match.")
		return nil
	}

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
	launchModulesDir := filepath.Join(m.launchLayer.Root, "node_modules")

	m.logger.SubsequentLine("Writing profile.d/NODE_PATH")
	if err := m.launchLayer.WriteProfile("NODE_PATH", fmt.Sprintf("export NODE_PATH=%s", launchModulesDir)); err != nil {
		return fmt.Errorf("failed to write NODE_PATH in the launch layer: %v", err)
	}

	return nil
}
