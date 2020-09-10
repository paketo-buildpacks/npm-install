package npminstall

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
)

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(pexec.Execution) (err error)
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
	executable Executable
	summer     Summer
	logger     scribe.Logger
}

func NewBuildProcessResolver(executable Executable, summer Summer, logger scribe.Logger) BuildProcessResolver {
	return BuildProcessResolver{
		executable: executable,
		summer:     summer,
		logger:     logger,
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

	wasItFound := map[bool]string{
		true:  "Found",
		false: "Not found",
	}

	inputsMap := scribe.FormattedMap{
		"package-lock.json": wasItFound[locked],
		"node_modules":      wasItFound[vendored],
		"npm-cache":         wasItFound[cached],
	}

	r.logger.Subprocess("Process inputs:")
	r.logger.Action("%s", inputsMap)
	r.logger.Break()

	switch {
	case !locked && vendored, locked && vendored && !cached:
		r.logger.Subprocess("Selected NPM build process: 'npm rebuild'")
		r.logger.Break()
		return NewRebuildBuildProcess(r.executable, r.summer, scribe.NewLogger(os.Stdout)), nil

	case !locked && !vendored:
		r.logger.Subprocess("Selected NPM build process: 'npm install'")
		r.logger.Break()
		return NewInstallBuildProcess(r.executable, scribe.NewLogger(os.Stdout)), nil

	default:
		r.logger.Subprocess("Selected NPM build process: 'npm ci'")
		r.logger.Break()
		return NewCIBuildProcess(r.executable, r.summer, scribe.NewLogger(os.Stdout)), nil
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
