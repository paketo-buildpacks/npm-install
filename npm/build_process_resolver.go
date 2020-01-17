package npm

import (
	"bytes"
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

//go:generate faux --interface BuildProcess --output fakes/build_process.go
type BuildProcess interface {
	Run(layerDir, cacheDir, workingDir string) error
}

func (r BuildProcessResolver) Resolve(workingDir, cacheDir string) (BuildProcess, error) {
	nodeModulesPath := filepath.Join(workingDir, "node_modules")
	vendored, err := fileExists(nodeModulesPath)
	if err != nil {
		return nil, err
	}

	packageLockPath := filepath.Join(workingDir, "package-lock.json")
	locked, err := fileExists(packageLockPath)
	if err != nil {
		return nil, err
	}

	npmCachePath := filepath.Join(workingDir, "npm-cache")
	cached, err := fileExists(npmCachePath)
	if err != nil {
		return nil, err
	}

	if cached {
		err := fs.Move(npmCachePath, filepath.Join(cacheDir, "npm-cache"))
		if err != nil {
			return nil, err
		}
	}

	switch {
	case !locked && vendored, locked && vendored && !cached:
		return RebuildBuildProcess{
			executable:    r.executable,
			scriptsParser: r.scriptsParser,
		}, nil

	case !locked && !vendored:
		return InstallBuildProcess{
			executable: r.executable,
		}, nil

	default:
		return CIBuildProcess{
			executable: r.executable,
		}, nil
	}
}

type RebuildBuildProcess struct {
	executable    Executable
	scriptsParser ScriptsParser
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

func fileExists(path string) (bool, error) {
	exists := true
	_, err := os.Stat(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return false, err
		}
		exists = false
	}
	return exists, nil
}
