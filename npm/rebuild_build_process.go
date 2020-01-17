package npm

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/packit/fs"
	"github.com/cloudfoundry/packit/pexec"
)

type RebuildBuildProcess struct {
	executable    Executable
	scriptsParser ScriptsParser
}

func NewRebuildBuildProcess(executable Executable, scriptsParser ScriptsParser) RebuildBuildProcess {
	return RebuildBuildProcess{
		executable:    executable,
		scriptsParser: scriptsParser,
	}
}

func (r RebuildBuildProcess) Run(layerDir, cacheDir, workingDir string) error {
	buffer := bytes.NewBuffer(nil)
	_, _, err := r.executable.Execute(pexec.Execution{
		Args:   []string{"list"},
		Dir:    workingDir,
		Stdout: buffer,
		Stderr: buffer,
	})
	if err != nil {
		return fmt.Errorf("vendored node_modules have unmet dependencies:\n%s\n%w", buffer, err)
	}

	err = fs.Move(filepath.Join(workingDir, "node_modules"), filepath.Join(layerDir, "node_modules"))
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(layerDir, "node_modules"), filepath.Join(workingDir, "node_modules"))
	if err != nil {
		return err
	}

	scripts, err := r.scriptsParser.ParseScripts(filepath.Join(workingDir, "package.json"))
	if err != nil {
		return fmt.Errorf("failed to parse package.json: %s", err)
	}

	if _, exists := scripts["preinstall"]; exists {
		_, _, err = r.executable.Execute(pexec.Execution{
			Args: []string{"run-script", "preinstall"},
			Dir:  workingDir,
		})

		if err != nil {
			return fmt.Errorf("preinstall script failed on rebuild: %s", err)
		}
	}

	_, _, err = r.executable.Execute(pexec.Execution{
		Args: []string{"rebuild", fmt.Sprintf("--nodedir=%s", os.Getenv("NODE_HOME"))},
		Dir:  workingDir,
	})
	if err != nil {
		return fmt.Errorf("npm rebuild failed: %s", err)
	}

	if _, exists := scripts["postinstall"]; exists {
		_, _, err = r.executable.Execute(pexec.Execution{
			Args: []string{"run-script", "postinstall"},
			Dir:  workingDir,
		})

		if err != nil {
			return fmt.Errorf("postinstall script failed on rebuild: %s", err)
		}
	}

	return nil
}
