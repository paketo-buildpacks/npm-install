package npminstall

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
)

type CIBuildProcess struct {
	executable  Executable
	summer      Summer
	environment EnvironmentConfig
	logger      scribe.Logger
}

func NewCIBuildProcess(executable Executable, summer Summer, environment EnvironmentConfig, logger scribe.Logger) CIBuildProcess {
	return CIBuildProcess{
		executable:  executable,
		summer:      summer,
		environment: environment,
		logger:      logger,
	}
}

func (r CIBuildProcess) ShouldRun(workingDir string, metadata map[string]interface{}) (bool, string, error) {
	cachedNodeVersion, err := cacheExecutableResponse(
		r.executable,
		[]string{"get", "user-agent"},
		workingDir,
		r.logger)
	if err != nil {
		return false, "", fmt.Errorf("failed to execute npm get user-agent: %w", err)
	}
	defer os.Remove(cachedNodeVersion)

	sum, err := r.summer.Sum(
		filepath.Join(workingDir, "package.json"),
		filepath.Join(workingDir, "package-lock.json"),
		cachedNodeVersion)
	if err != nil {
		return false, "", err
	}

	cacheSha, ok := metadata["cache_sha"].(string)
	if !ok || sum != cacheSha {
		return true, sum, nil
	}

	return false, "", nil
}

func (r CIBuildProcess) Run(modulesDir, cacheDir, workingDir string) error {
	err := os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)
	if err != nil {
		return err
	}

	err = fs.Move(filepath.Join(workingDir, "node_modules"), filepath.Join(modulesDir, "node_modules"))
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(modulesDir, "node_modules"), filepath.Join(workingDir, "node_modules"))
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer(nil)
	args := []string{"ci", "--unsafe-perm", "--cache", cacheDir}

	r.logger.Subprocess("Running 'npm %s'", strings.Join(args, " "))
	err = r.executable.Execute(pexec.Execution{
		Args:   args,
		Dir:    workingDir,
		Stdout: buffer,
		Stderr: buffer,
		Env: append(
			os.Environ(),
			fmt.Sprintf("NPM_CONFIG_LOGLEVEL=%s", r.environment.GetValue("NPM_CONFIG_LOGLEVEL")),
		),
	})
	if err != nil {
		r.logger.Subprocess("%s", buffer.String())
		return fmt.Errorf("npm ci failed: %w", err)
	}

	return nil
}
