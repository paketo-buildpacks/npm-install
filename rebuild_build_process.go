package npminstall

import (
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

func (r RebuildBuildProcess) ShouldRun(workingDir string, metadata map[string]interface{}, npmrcPath string) (bool, string, error) {
	cachedNodeVersion, err := cacheExecutableResponse(
		r.executable,
		[]string{"get", "user-agent"},
		workingDir,
		npmrcPath,
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

func (r RebuildBuildProcess) Run(modulesDir, cacheDir, workingDir, npmrcPath string, launch bool) error {
	environment := os.Environ()
	if npmrcPath != "" {
		environment = append(environment, fmt.Sprintf("NPM_CONFIG_GLOBALCONFIG=%s", npmrcPath))
	}

	err := r.executable.Execute(pexec.Execution{
		Args:   []string{"list"},
		Dir:    workingDir,
		Env:    environment,
		Stdout: r.logger.ActionWriter,
		Stderr: r.logger.ActionWriter,
	})
	if err != nil {
		return fmt.Errorf("vendored node_modules have unmet dependencies: npm list failed: %w", err)
	}

	args := []string{"run-script", "preinstall", "--if-present"}
	r.logger.Subprocess("Running 'npm %s'", strings.Join(args, " "))
	err = r.executable.Execute(pexec.Execution{
		Args:   args,
		Dir:    workingDir,
		Env:    environment,
		Stdout: r.logger.ActionWriter,
		Stderr: r.logger.ActionWriter,
	})

	if err != nil {
		return fmt.Errorf("preinstall script failed on rebuild: %s", err)
	}

	env := environment
	if value, ok := r.environment.Lookup("NPM_CONFIG_LOGLEVEL"); ok {
		env = append(env, fmt.Sprintf("NPM_CONFIG_LOGLEVEL=%s", value))
	}

	if !launch {
		env = append(env, "NODE_ENV=development")
	}

	nodeHome, _ := r.environment.Lookup("NODE_HOME")
	args = []string{"rebuild", fmt.Sprintf("--nodedir=%s", nodeHome)}
	r.logger.Subprocess("Running 'npm %s'", strings.Join(args, " "))
	err = r.executable.Execute(pexec.Execution{
		Args:   args,
		Dir:    workingDir,
		Stdout: r.logger.ActionWriter,
		Stderr: r.logger.ActionWriter,
		Env:    env,
	})
	if err != nil {
		return fmt.Errorf("npm rebuild failed: %s", err)
	}

	args = []string{"run-script", "postinstall", "--if-present"}
	r.logger.Subprocess("Running 'npm %s'", strings.Join(args, " "))
	err = r.executable.Execute(pexec.Execution{
		Args:   args,
		Dir:    workingDir,
		Env:    environment,
		Stdout: r.logger.ActionWriter,
		Stderr: r.logger.ActionWriter,
	})

	if err != nil {
		return fmt.Errorf("postinstall script failed on rebuild: %s", err)
	}

	_, err = os.Stat(filepath.Join(workingDir, "node_modules"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("unable to stat node_modules in working directory: %w", err)
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
