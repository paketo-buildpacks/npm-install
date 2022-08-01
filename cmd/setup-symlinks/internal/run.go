package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Run(executablePath, appDir, tmpDir string) error {
	fname := strings.Split(executablePath, "/")
	layerPath := filepath.Join(fname[:len(fname)-2]...)
	if filepath.IsAbs(executablePath) {
		layerPath = fmt.Sprintf("/%s", layerPath)
	}

	err := os.RemoveAll(filepath.Join(tmpDir, "node_modules"))
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(layerPath, "node_modules"), filepath.Join(tmpDir, "node_modules"))
	if err != nil {
		return err
	}

	return nil
}
