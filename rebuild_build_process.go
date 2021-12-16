package npminstall

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

type RebuildBuildProcess struct {
	executable  Executable
	summer      Summer
	environment EnvironmentConfig
	logger      scribe.Logger
}

func NewRebuildBuildProcess(executable Executable, summer Summer, environment EnvironmentConfig, logger scribe.Logger) RebuildBuildProcess {
	return RebuildBuildProcess{
		executable:  executable,
		summer:      summer,
		environment: environment,
		logger:      logger,
	}
}

func (r RebuildBuildProcess) ShouldRun(workingDir string, metadata map[string]interface{}) (bool, string, error) {
	cachedNodeVersion, err := cacheExecutableResponse(
		r.executable,
		[]string{"get", "user-agent"},
		workingDir,
		r.logger)
	if err != nil {
		return false, "", fmt.Errorf("failed to execute npm get user-agent: %w", err)
	}
	defer os.Remove(cachedNodeVersion)

	sum, err := r.summer.Sum(filepath.Join(workingDir, "node_modules"), cachedNodeVersion)
	if err != nil {
		return false, "", err
	}

	cacheSha, ok := metadata["cache_sha"].(string)
	if !ok || sum != cacheSha {
		return true, sum, nil
	}

	return false, "", nil
}

func (r RebuildBuildProcess) Run(modulesDir, cacheDir, workingDir string) error {
	buffer := bytes.NewBuffer(nil)
	err := r.executable.Execute(pexec.Execution{
		Args:   []string{"list"},
		Dir:    workingDir,
		Stdout: buffer,
		Stderr: buffer,
	})
	if err != nil {
		r.logger.Subprocess("%s", buffer.String())
		return fmt.Errorf("vendored node_modules have unmet dependencies: npm list failed: %w", err)
	}

	args := []string{"run-script", "preinstall", "--if-present"}
	r.logger.Subprocess("Running 'npm %s'", strings.Join(args, " "))
	err = r.executable.Execute(pexec.Execution{
		Args:   args,
		Dir:    workingDir,
		Stdout: buffer,
		Stderr: buffer,
	})

	if err != nil {
		r.logger.Subprocess("%s", buffer.String())
		return fmt.Errorf("preinstall script failed on rebuild: %s", err)
	}

	args = []string{"rebuild", fmt.Sprintf("--nodedir=%s", os.Getenv("NODE_HOME"))}
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
		return fmt.Errorf("npm rebuild failed: %s", err)
	}

	args = []string{"run-script", "postinstall", "--if-present"}
	r.logger.Subprocess("Running 'npm %s'", strings.Join(args, " "))
	err = r.executable.Execute(pexec.Execution{
		Args:   args,
		Dir:    workingDir,
		Stdout: buffer,
		Stderr: buffer,
	})

	if err != nil {
		r.logger.Subprocess("%s", buffer.String())
		return fmt.Errorf("postinstall script failed on rebuild: %s", err)
	}

	_, err = os.Stat(filepath.Join(workingDir, "node_modules"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		} else {
			return fmt.Errorf("unable to stat node_modules in working directory: %w", err)
		}
	}

	err = fs.Move(filepath.Join(workingDir, "node_modules"), filepath.Join(modulesDir, "node_modules"))
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(modulesDir, "node_modules"), filepath.Join(workingDir, "node_modules"))
	if err != nil {
		return err
	}

	return nil
}
