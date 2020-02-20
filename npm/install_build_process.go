package npm

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/packit/pexec"
	"github.com/cloudfoundry/packit/scribe"
)

func NewInstallBuildProcess(executable Executable, logger scribe.Logger) InstallBuildProcess {
	return InstallBuildProcess{
		executable: executable,
		logger:     logger,
	}
}

type InstallBuildProcess struct {
	executable Executable
	logger     scribe.Logger
}

func (r InstallBuildProcess) ShouldRun(workingDir string, metadata map[string]interface{}) (bool, string, error) {
	return true, "", nil
}

func (r InstallBuildProcess) Run(modulesDir, cacheDir, workingDir string) error {
	r.logger.Subprocess("Running 'npm install'")

	err := os.Mkdir(filepath.Join(modulesDir, "node_modules"), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(modulesDir, "node_modules"), filepath.Join(workingDir, "node_modules"))
	if err != nil {
		return err
	}

	buffer := bytes.NewBuffer(nil)
	err = r.executable.Execute(pexec.Execution{
		Args:   []string{"install", "--unsafe-perm", "--cache", cacheDir},
		Dir:    workingDir,
		Stdout: buffer,
		Stderr: buffer,
		Env:    append(os.Environ(), "NPM_CONFIG_PRODUCTION=true", "NPM_CONFIG_LOGLEVEL=error"),
	})
	if err != nil {
		r.logger.Subprocess("%s", buffer.String())
		return fmt.Errorf("npm install failed: %w", err)
	}

	return nil
}
