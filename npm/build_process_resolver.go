package npm

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/packit/fs"
	"github.com/cloudfoundry/packit/pexec"
)

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(pexec.Execution) (stdout, stderr string, err error)
}

type BuildProcessResolver struct {
	executable Executable
}

func NewBuildProcessResolver(executable Executable) BuildProcessResolver {
	return BuildProcessResolver{
		executable: executable,
	}
}

type BuildProcess func(layerDir, workingDir string) error

func (r BuildProcessResolver) Resolve(workingDir string) (BuildProcess, error) {
	vendored := true
	_, err := os.Stat(filepath.Join(workingDir, "node_modules"))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}

		vendored = false
	}

	switch {
	case vendored:
		return r.rebuild, nil
	default:
		return r.install, nil
	}
}

func (r BuildProcessResolver) install(layerDir, workingDir string) error {
	err := os.Mkdir(filepath.Join(layerDir, "node_modules"), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(layerDir, "node_modules"), filepath.Join(workingDir, "node_modules"))
	if err != nil {
		return err
	}

	_, _, err = r.executable.Execute(pexec.Execution{
		Args: []string{"install"},
		Dir:  workingDir,
	})
	if err != nil {
		return err
	}

	return nil
}

func (r BuildProcessResolver) rebuild(layerDir, workingDir string) error {
	err := fs.Move(filepath.Join(workingDir, "node_modules"), filepath.Join(layerDir, "node_modules"))
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(layerDir, "node_modules"), filepath.Join(workingDir, "node_modules"))
	if err != nil {
		return err
	}

	_, _, err = r.executable.Execute(pexec.Execution{
		Args: []string{"rebuild"},
		Dir:  workingDir,
	})
	if err != nil {
		return err
	}

	_, _, err = r.executable.Execute(pexec.Execution{
		Args: []string{"install"},
		Dir:  workingDir,
	})
	if err != nil {
		return err
	}

	return nil
}
