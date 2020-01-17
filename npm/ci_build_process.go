package npm

import (
	"os"
	"path/filepath"

	"github.com/cloudfoundry/packit/fs"
	"github.com/cloudfoundry/packit/pexec"
)

type CIBuildProcess struct {
	executable Executable
}

func NewCIBuildProcess(executable Executable) CIBuildProcess {
	return CIBuildProcess{
		executable: executable,
	}
}

func (r CIBuildProcess) Run(layerDir, cacheDir, workingDir string) error {
	err := os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)
	if err != nil {
		return err
	}

	err = fs.Move(filepath.Join(workingDir, "node_modules"), filepath.Join(layerDir, "node_modules"))
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(layerDir, "node_modules"), filepath.Join(workingDir, "node_modules"))
	if err != nil {
		return err
	}

	_, _, err = r.executable.Execute(pexec.Execution{
		Args: []string{"ci", "--unsafe-perm", "--cache", cacheDir},
		Dir:  workingDir,
	})
	if err != nil {
		return err
	}

	return nil
}
