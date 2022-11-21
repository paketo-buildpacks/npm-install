package npminstall

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(pexec.Execution) (err error)
}

//go:generate faux --interface BuildProcess --output fakes/build_process.go
type BuildProcess interface {
	ShouldRun(workingDir string, metadata map[string]interface{}, npmrcPath string) (run bool, sha string, err error)
	Run(modulesDir, cacheDir, workingDir, npmrcPath string, launch bool) error
}

//go:generate faux --interface Summer --output fakes/summer.go
type Summer interface {
	Sum(paths ...string) (string, error)
}

//go:generate faux --interface EnvironmentConfig --output fakes/environment_config.go
type EnvironmentConfig interface {
	GetValue(key string) string
}

type BuildProcessResolver struct {
	executable  Executable
	summer      Summer
	environment EnvironmentConfig
	logger      scribe.Logger
}

func NewBuildProcessResolver(executable Executable, summer Summer, environment EnvironmentConfig, logger scribe.Logger) BuildProcessResolver {
	return BuildProcessResolver{
		executable:  executable,
		summer:      summer,
		environment: environment,
		logger:      logger,
	}
}

func (r BuildProcessResolver) Resolve(workingDir string) (BuildProcess, bool, error) {
	nodeModulesPath := filepath.Join(workingDir, "node_modules")
	vendored, err := fs.Exists(nodeModulesPath)
	if err != nil {
		return nil, false, err
	}

	packageLockPath := filepath.Join(workingDir, "package-lock.json")
	locked, err := fs.Exists(packageLockPath)
	if err != nil {
		return nil, false, err
	}

	npmCachePath := filepath.Join(workingDir, "npm-cache")
	cached, err := fs.Exists(npmCachePath)
	if err != nil {
		return nil, false, err
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
		return NewRebuildBuildProcess(r.executable, r.summer, r.environment, scribe.NewLogger(os.Stdout)), cached, nil

	case !locked && !vendored:
		r.logger.Subprocess("Selected NPM build process: 'npm install'")
		r.logger.Break()
		return NewInstallBuildProcess(r.executable, r.environment, scribe.NewLogger(os.Stdout)), cached, nil

	default:
		r.logger.Subprocess("Selected NPM build process: 'npm ci'")
		r.logger.Break()
		return NewCIBuildProcess(r.executable, r.summer, r.environment, scribe.NewLogger(os.Stdout)), cached, nil
	}
}

// cacheExecutableResponse writes the output of a successfully executed command
// to a tmp file and returns the file location and possibly and error
func cacheExecutableResponse(executable Executable, args []string, workingDir string, npmrcPath string, logger scribe.Logger) (string, error) {
	stdout := bytes.NewBuffer(nil)
	stderr := bytes.NewBuffer(nil)
	var environment []string
	if npmrcPath != "" {
		environment = append(os.Environ(), fmt.Sprintf("NPM_CONFIG_GLOBALCONFIG=%s", npmrcPath))
	}
	err := executable.Execute(pexec.Execution{
		Args:   args,
		Dir:    workingDir,
		Env:    environment,
		Stdout: stdout,
		Stderr: stderr,
	})
	if err != nil {
		logger.Subprocess("error: %s", stderr.String())
		return "", err
	}

	tmpFile, err := os.CreateTemp(workingDir, "executable_response")
	if err != nil {
		logger.Subprocess("error: %s", err)
		return "", err
	}

	err = os.WriteFile(tmpFile.Name(), stdout.Bytes(), 0644)
	if err != nil {
		logger.Subprocess("error: %s", err)
		return "", err
	}

	return tmpFile.Name(), nil
}
