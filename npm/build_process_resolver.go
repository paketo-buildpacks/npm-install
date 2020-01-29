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

//go:generate faux --interface BuildProcess --output fakes/build_process.go
type BuildProcess interface {
	ShouldRun(workingDir string, metadata map[string]interface{}) (run bool, sha string, err error)
	Run(modulesDir, cacheDir, workingDir string) error
}

//go:generate faux --interface Summer --output fakes/summer.go
type Summer interface {
	Sum(path string) (string, error)
}

type BuildProcessResolver struct {
	executable    Executable
	scriptsParser ScriptsParser
	summer        Summer
}

func NewBuildProcessResolver(executable Executable, scriptsParser ScriptsParser, summer Summer) BuildProcessResolver {
	return BuildProcessResolver{
		executable:    executable,
		scriptsParser: scriptsParser,
		summer:        summer,
	}
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
		return NewRebuildBuildProcess(r.executable, r.scriptsParser, r.summer), nil

	case !locked && !vendored:
		return NewInstallBuildProcess(r.executable), nil

	default:
		return NewCIBuildProcess(r.executable, r.summer), nil
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
