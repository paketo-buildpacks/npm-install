package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Run(executablePath, appDir string) error {
	fname := strings.Split(executablePath, "/")
	layerPath := filepath.Join(fname[:len(fname)-2]...)
	if filepath.IsAbs(executablePath) {
		layerPath = fmt.Sprintf("/%s", layerPath)
	}

	linkPath, err := os.Readlink(filepath.Join(appDir, "node_modules"))
	if err != nil {
		return err
	}

	linkPath, err = filepath.Abs(linkPath)
	if err != nil {
		return err
	}

	return createSymlink(filepath.Join(layerPath, "node_modules"), linkPath)
}

func createSymlink(target, source string) error {
	err := os.RemoveAll(source)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(source), os.ModePerm)
	if err != nil {
		return err
	}

	return os.Symlink(target, source)
}
