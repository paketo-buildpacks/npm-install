package npminstall

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/paketo-buildpacks/packit/v2/pexec"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

func NewPruneBuildProcess(executable Executable, environment EnvironmentConfig, logger scribe.Logger) PruneBuildProcess {
	return PruneBuildProcess{
		executable:  executable,
		environment: environment,
		logger:      logger,
	}
}

type PruneBuildProcess struct {
	executable  Executable
	environment EnvironmentConfig
	logger      scribe.Logger
}

func (r PruneBuildProcess) ShouldRun(workingDir string, metadata map[string]interface{}, npmrcPath string) (bool, string, error) {
	return true, "", nil
}

func (r PruneBuildProcess) Run(modulesDir, cacheDir, workingDir, npmrcPath string, launch bool) error {
	buffer := bytes.NewBuffer(nil)
	args := []string{"prune"}
	environment := append(os.Environ(), fmt.Sprintf("NPM_CONFIG_LOGLEVEL=%s", r.environment.GetValue("NPM_CONFIG_LOGLEVEL")))
	if npmrcPath != "" {
		environment = append(environment, fmt.Sprintf("NPM_CONFIG_GLOBALCONFIG=%s", npmrcPath))
	}

	r.logger.Subprocess("Running 'npm %s'", strings.Join(args, " "))
	err := r.executable.Execute(pexec.Execution{
		Args:   args,
		Dir:    workingDir,
		Stdout: buffer,
		Stderr: buffer,
		Env:    environment,
	})
	if err != nil {
		r.logger.Subprocess("%s", buffer.String())
		return fmt.Errorf("npm install failed: %w", err)
	}

	return nil
}
