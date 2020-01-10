package npm

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/packit/fs"
	"github.com/cloudfoundry/packit/pexec"
)

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(pexec.Execution) (stdout, stderr string, err error)
}

//go:generate faux --interface ScriptsParser --output fakes/scripts_parser.go
type ScriptsParser interface {
	ParseScripts(path string) (scriptsMap map[string]string, err error)
}

type BuildProcessResolver struct {
	executable    Executable
	scriptsParser ScriptsParser
}

func NewBuildProcessResolver(executable Executable, scriptsParser ScriptsParser) BuildProcessResolver {
	return BuildProcessResolver{
		executable:    executable,
		scriptsParser: scriptsParser,
	}
}

type BuildProcess func(layerDir, workingDir string) error

func (r BuildProcessResolver) Resolve(workingDir string) (BuildProcess, error) {
	vendored := true
	_, err := os.Stat(filepath.Join(workingDir, "node_modules"))
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}

		vendored = false
	}

	switch {
	case vendored:
		return r.rebuild, nil
	default:
		return r.install, nil
	}
}

func (r BuildProcessResolver) install(layerDir, workingDir string) error {
	err := os.Mkdir(filepath.Join(layerDir, "node_modules"), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(layerDir, "node_modules"), filepath.Join(workingDir, "node_modules"))
	if err != nil {
		return err
	}

	_, _, err = r.executable.Execute(pexec.Execution{
		Args: []string{"install"},
		Dir:  workingDir,
	})
	if err != nil {
		return err
	}

	return nil
}

func (r BuildProcessResolver) rebuild(layerDir, workingDir string) error {
	err := fs.Move(filepath.Join(workingDir, "node_modules"), filepath.Join(layerDir, "node_modules"))
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
			Args:   []string{"run-script", "preinstall"},
			Dir:    workingDir,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
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
			Args:   []string{"run-script", "postinstall"},
			Dir:    workingDir,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		})

		if err != nil {
			return fmt.Errorf("postinstall script failed on rebuild: %s", err)
		}
	}

	return nil
}
