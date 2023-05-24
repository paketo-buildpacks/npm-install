package npminstall

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2/fs"
)

type Lockfile struct {
	Packages map[string]struct {
		Resolved string `json:"resolved"`
		Link     bool   `json:"link"`
	} `json:"packages"`
}

type LinkedModuleResolver struct {
	linker Symlinker
}

func NewLinkedModuleResolver(linker Symlinker) LinkedModuleResolver {
	return LinkedModuleResolver{
		linker: linker,
	}
}

func (r LinkedModuleResolver) ParseLockfile(lockfilePath string) (Lockfile, error) {
	file, err := os.Open(lockfilePath)
	if err != nil {
		return Lockfile{}, fmt.Errorf(`failed to open "package-lock.json": %w`, err)
	}
	defer file.Close()

	parsedLockfile := Lockfile{}

	err = json.NewDecoder(file).Decode(&parsedLockfile)
	if err != nil {
		return Lockfile{}, fmt.Errorf(`failed to parse "package-lock.json": %w`, err)
	}

	return parsedLockfile, nil
}

func (r LinkedModuleResolver) Copy(lockfilePath, sourceLayerPath, targetLayerPath string) error {

	lockfile, err := r.ParseLockfile(lockfilePath)
	if err != nil {
		panic(err)
	}

	for _, pkg := range lockfile.Packages {
		if pkg.Link {
			source := filepath.Join(sourceLayerPath, pkg.Resolved)
			destination := filepath.Join(targetLayerPath, pkg.Resolved)

			err = os.MkdirAll(filepath.Dir(destination), os.ModePerm)
			if err != nil {
				return fmt.Errorf("failed to setup linked module directory scaffolding: %w", err)
			}

			err = fs.Copy(source, destination)
			if err != nil {
				return fmt.Errorf("failed to copy linked module directory to layer path: %w", err)
			}
		}
	}

	return nil

}

func (r LinkedModuleResolver) Resolve(lockfilePath, layerPath string) error {

	lockfile, err := r.ParseLockfile(lockfilePath)
	if err != nil {
		return err
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
