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

//go:generate faux --interface ScriptsParser --output fakes/scripts_parser.go
type ScriptsParser interface {
	ParseScripts(path string) (scripts map[string]string, err error)
}

type BuildProcessResolver struct {
	executable    Executable
	scriptsParser ScriptsParser
}

func NewBuildProcessResolver(executable Executable, scriptsParser ScriptsParser) BuildProcessResolver {
	return BuildProcessResolver{
		executable:    executable,
		scriptsParser: scriptsParser,
	}
}

//go:generate faux --interface BuildProcess --output fakes/build_process.go
type BuildProcess interface {
	Run(layerDir, cacheDir, workingDir string) error
}

func (r BuildProcessResolver) Resolve(workingDir, cacheDir string) (BuildProcess, error) {
	nodeModulesPath := filepath.Join(workingDir, "node_modules")
	vendored, err := fileExists(nodeModulesPath)
	if err != nil {
		return nil, err
	}

	packageLockPath := filepath.Join(workingDir, "package-lock.json")
	locked, err := fileExists(packageLockPath)
	if err != nil {
		return nil, err
	}

	npmCachePath := filepath.Join(workingDir, "npm-cache")
	cached, err := fileExists(npmCachePath)
	if err != nil {
		return nil, err
	}

	if cached {
		err := fs.Move(npmCachePath, filepath.Join(cacheDir, "npm-cache"))
		if err != nil {
			return nil, err
		}
	}

	switch {
	case !locked && vendored, locked && vendored && !cached:
		return RebuildBuildProcess{
			executable:    r.executable,
			scriptsParser: r.scriptsParser,
		}, nil

	case !locked && !vendored:
		return InstallBuildProcess{
			executable: r.executable,
		}, nil

	default:
		return CIBuildProcess{
			executable: r.executable,
		}, nil
	}
}

func fileExists(path string) (bool, error) {
	exists := true
	_, err := os.Stat(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		exists = false
	}
	return exists, nil
}
