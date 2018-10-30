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

func (n *NPM) InstallToLayer(srcLayer, dstLayer string) error {

	srcPackageJsonPath := filepath.Join(srcLayer, "package.json")
	_, err := os.Stat(srcPackageJsonPath)
	if err != nil {
		return fmt.Errorf("package.json file not found at %s with error %s", srcPackageJsonPath, err)
	}

	if err = n.Runner.Run(
		srcLayer,
		"install",
		"--unsafe-perm",
		"--cache",
		fmt.Sprintf("%s/npm-cache", srcLayer),
	); err != nil {
		return err
	}

	return nil
}

func (n *NPM) CleanAndCopyToDst(src, dst string) error {
	if err := os.RemoveAll(dst); err != nil {
		return fmt.Errorf("failed to remove modules in %s : %v", dst, err)
	}

	if err := n.copyModules(src, dst); err != nil {
		return fmt.Errorf("failed to copy the src modules from %s to %s %v", src, dst, err)
	}
	return nil
}

func (n *NPM) RebuildLayer(srcLayer, dstLayer string) error {
	srcPackageJsonPath := filepath.Join(srcLayer, "package.json")
	_, err := os.Stat(srcPackageJsonPath)
	if err != nil {
		return fmt.Errorf("package.json file not found at %s with error %s", srcPackageJsonPath, err)
	}

	srcModulesDir := filepath.Join(srcLayer, "node_modules")
	dstModulesDir := filepath.Join(dstLayer, "node_modules")

	if err := n.Runner.Run(dstLayer, "rebuild"); err != nil {
		return err
	}

	if err := n.CleanAndCopyToDst(srcModulesDir, dstModulesDir); err != nil {
		return fmt.Errorf("failed to rebuild : %v", err)
	}
	return nil
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
