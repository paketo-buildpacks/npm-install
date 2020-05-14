package npm

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/paketo-buildpacks/packit/scribe"
)

type RebuildBuildProcess struct {
	executable Executable
	summer     Summer
	logger     scribe.Logger
}

func NewRebuildBuildProcess(executable Executable, summer Summer, logger scribe.Logger) RebuildBuildProcess {
	return RebuildBuildProcess{
		executable: executable,
		summer:     summer,
		logger:     logger,
	}
}

func (r RebuildBuildProcess) ShouldRun(workingDir string, metadata map[string]interface{}) (bool, string, error) {
	sum, err := r.summer.Sum(filepath.Join(workingDir, "node_modules"))
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
	r.logger.Subprocess("Running 'npm rebuild'")

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

	err = fs.Move(filepath.Join(workingDir, "node_modules"), filepath.Join(modulesDir, "node_modules"))
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(modulesDir, "node_modules"), filepath.Join(workingDir, "node_modules"))
	if err != nil {
		return err
	}

	err = r.executable.Execute(pexec.Execution{
		Args:   []string{"run-script", "preinstall", "--if-present"},
		Dir:    workingDir,
		Stdout: buffer,
		Stderr: buffer,
	})

	if err != nil {
		r.logger.Subprocess("%s", buffer.String())
		return fmt.Errorf("preinstall script failed on rebuild: %s", err)
	}

	err = r.executable.Execute(pexec.Execution{
		Args:   []string{"rebuild", fmt.Sprintf("--nodedir=%s", os.Getenv("NODE_HOME"))},
		Dir:    workingDir,
		Env:    append(os.Environ(), "NPM_CONFIG_PRODUCTION=true", "NPM_CONFIG_LOGLEVEL=error"),
		Stdout: buffer,
		Stderr: buffer,
	})
	if err != nil {
		r.logger.Subprocess("%s", buffer.String())
		return fmt.Errorf("npm rebuild failed: %s", err)
	}

	err = r.executable.Execute(pexec.Execution{
		Args:   []string{"run-script", "postinstall", "--if-present"},
		Dir:    workingDir,
		Stdout: buffer,
		Stderr: buffer,
	})

	if err != nil {
		r.logger.Subprocess("%s", buffer.String())
		return fmt.Errorf("postinstall script failed on rebuild: %s", err)
	}

	return nil
}
