package npm

import "github.com/cloudfoundry/packit/pexec"

//go:generate faux --interface Executable --output fakes/executable.go
type Executable interface {
	Execute(pexec.Execution) (stdout, stderr string, err error)
}

type NodePackageManager struct {
	executable Executable
}

func NewNodePackageManager(executable Executable) NodePackageManager {
	return NodePackageManager{
		executable: executable,
	}
}

func (m NodePackageManager) Install(dir string) error {
	_, _, err := m.executable.Execute(pexec.Execution{
		Args: []string{"install"},
		Dir:  dir,
	})
	if err != nil {
		return err
	}

	return nil
}
