package npminstall

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2/fs"
)

type LinkedModuleResolver struct {
	linker Symlinker
}

func NewLinkedModuleResolver(linker Symlinker) LinkedModuleResolver {
	return LinkedModuleResolver{
		linker: linker,
	}
}

func (r LinkedModuleResolver) Resolve(lockfilePath, layerPath string) error {
	file, err := os.Open(lockfilePath)
	if err != nil {
		return fmt.Errorf("failed to open \"package-lock.json\": %w", err)
	}
	defer file.Close()

	var lockfile struct {
		Packages map[string]struct {
			Resolved string `json:"resolved"`
			Link     bool   `json:"link"`
		} `json:"packages"`
	}
	err = json.NewDecoder(file).Decode(&lockfile)
	if err != nil {
		return fmt.Errorf("failed to parse \"package-lock.json\": %w", err)
	}

	dir := filepath.Dir(lockfilePath)
	for _, pkg := range lockfile.Packages {
		if pkg.Link {
			source := filepath.Join(dir, pkg.Resolved)
			destination := filepath.Join(layerPath, pkg.Resolved)

			err = os.MkdirAll(filepath.Dir(destination), os.ModePerm)
			if err != nil {
				return fmt.Errorf("failed to setup linked module directory scaffolding: %w", err)
			}

			err = fs.Copy(source, destination)
			if err != nil {
				return fmt.Errorf("failed to copy linked module directory to layer path: %w", err)
			}

			err = r.linker.WithPath(pkg.Resolved).Link(source, destination)
			if err != nil {
				return fmt.Errorf("failed to symlink linked module directory: %w", err)
			}
		}
	}

	return nil
}
