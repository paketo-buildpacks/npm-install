package npm

import (
	"os"
	"path/filepath"

	"github.com/cloudfoundry/packit/fs"
	"github.com/cloudfoundry/packit/pexec"
)

type CIBuildProcess struct {
	executable Executable
	summer     Summer
}

func NewCIBuildProcess(executable Executable, summer Summer) CIBuildProcess {
	return CIBuildProcess{
		executable: executable,
		summer:     summer,
	}
}

func (r CIBuildProcess) ShouldRun(workingDir string, metadata map[string]interface{}) (bool, string, error) {
	sum, err := r.summer.Sum(filepath.Join(workingDir, "package-lock.json"))
	if err != nil {
		return false, "", err
	}

	cacheSha, ok := metadata["cache_sha"].(string)
	if !ok || sum != cacheSha {
		return true, sum, nil
	}

	return false, "", nil
}

func (r CIBuildProcess) Run(modulesDir, cacheDir, workingDir string) error {
	err := os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm)
	if err != nil {
		return err
	}

	err = fs.Move(filepath.Join(workingDir, "node_modules"), filepath.Join(modulesDir, "node_modules"))
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(modulesDir, "node_modules"), filepath.Join(workingDir, "node_modules"))
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
