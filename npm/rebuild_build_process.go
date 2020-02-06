package npm

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/packit/fs"
	"github.com/cloudfoundry/packit/pexec"
	"github.com/cloudfoundry/packit/scribe"
)

type RebuildBuildProcess struct {
	executable    Executable
	scriptsParser ScriptsParser
	summer        Summer
	logger        scribe.Logger
}

func NewRebuildBuildProcess(executable Executable, scriptsParser ScriptsParser, summer Summer, logger scribe.Logger) RebuildBuildProcess {
	return RebuildBuildProcess{
		executable:    executable,
		scriptsParser: scriptsParser,
		summer:        summer,
		logger:        logger,
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
	buffer := bytes.NewBuffer(nil)
	_, _, err := r.executable.Execute(pexec.Execution{
		Args:   []string{"list"},
		Dir:    workingDir,
		Stdout: buffer,
		Stderr: buffer,
	})
	if err != nil {
		r.logger.Process("%s", buffer.String())
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

	scripts, err := r.scriptsParser.ParseScripts(filepath.Join(workingDir, "package.json"))
	if err != nil {
		return fmt.Errorf("failed to parse package.json: %s", err)
	}

	if _, exists := scripts["preinstall"]; exists {
		_, _, err = r.executable.Execute(pexec.Execution{
			Args:   []string{"run-script", "preinstall"},
			Dir:    workingDir,
			Stdout: buffer,
			Stderr: buffer,
		})

		if err != nil {
			r.logger.Process("%s", buffer.String())
			return fmt.Errorf("preinstall script failed on rebuild: %s", err)
		}
	}

	_, _, err = r.executable.Execute(pexec.Execution{
		Args:   []string{"rebuild", fmt.Sprintf("--nodedir=%s", os.Getenv("NODE_HOME"))},
		Dir:    workingDir,
		Env:    append(os.Environ(), "NPM_CONFIG_PRODUCTION=true", "NPM_CONFIG_LOGLEVEL=error"),
		Stdout: buffer,
		Stderr: buffer,
	})
	if err != nil {
		r.logger.Process("%s", buffer.String())
		return fmt.Errorf("npm rebuild failed: %s", err)
	}

	if _, exists := scripts["postinstall"]; exists {
		_, _, err = r.executable.Execute(pexec.Execution{
			Args:   []string{"run-script", "postinstall"},
			Dir:    workingDir,
			Stdout: buffer,
			Stderr: buffer,
		})

		if err != nil {
			r.logger.Process("%s", buffer.String())
			return fmt.Errorf("postinstall script failed on rebuild: %s", err)
		}
	}

	return nil
}
