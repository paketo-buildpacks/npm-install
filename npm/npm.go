package npm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libjavabuildpack"
	"github.com/cloudfoundry/npm-cnb/utils"
)

type NPM struct {
	Runner Runner
}

func NewNPM() *NPM {
	return &NPM{
		Runner: &npmCmd{},
	}
}

// Install node_mdules from a src layer to a target
func (n *NPM) InstallInLayer(src, dst string) error {
	if err := os.MkdirAll(dst, 0777); err != nil {
		return fmt.Errorf("failed to create directory %s : %v", dst, err)
	}

	appPackageJsonPath := filepath.Join(src, "package.json")
	cachePackageJsonPath := filepath.Join(dst, "package.json")
	if err := utils.CopyFile(appPackageJsonPath, cachePackageJsonPath); err != nil {
		return fmt.Errorf("failed to copy package.json : %v", err)
	}

	appPackageLockPath := filepath.Join(src, "package-lock.json")
	cachePackageLockPath := filepath.Join(dst, "package-lock.json")
	if err := utils.CopyFile(appPackageLockPath, cachePackageLockPath); err != nil {
		return fmt.Errorf("failed to copy package-lock.json : %v", err)
	}

	return n.Runner.Run(src,
		"--prefix",
		dst,
		"install",
		"--unsafe-perm",
		"--cache",
		fmt.Sprintf("%s/npm-cache", dst),
	)
}

func (n *NPM) CopyToDst(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("failed to remove modules in %s : %v", dst, err)
	}

	if err := n.copyModules(src, dst); err != nil {
		return fmt.Errorf("failed to copy the src modules from %s to %s %v", src, dst, err)
	}
	return nil
}

// RebuildLayer copies node_modules from a source layer to a target layer and
// runs a `npm rebuild`
func (n *NPM) RebuildLayer(srcLayer, dstLayer string) error {
	srcModulesDir := filepath.Join(srcLayer, "node_modules")
	dstModulesDir := filepath.Join(dstLayer, "node_modules")

	if err := n.CopyToDst(srcModulesDir, dstModulesDir); err != nil {
		return fmt.Errorf("failed to rebuild : %v", err)
	}

	return n.Runner.Run(dstLayer, "rebuild")
}

func (n *NPM) copyModules(src, dst string) error {
	if exist, err := libjavabuildpack.FileExists(dst); err != nil {
		return err
	} else if !exist {
		if err := os.MkdirAll(dst, 0777); err != nil {
			return err
		}
	}

	if err := utils.CopyDirectory(src, dst); err != nil {
		return err
	}

	return nil
}
