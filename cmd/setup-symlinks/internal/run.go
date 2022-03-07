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

	err := os.RemoveAll(filepath.Join(appDir, "node_modules"))
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(layerPath, "node_modules"), filepath.Join(appDir, "node_modules"))
	if err != nil {
		return err
	}

	return nil
}
