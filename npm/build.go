package npm

import (
	"os"
	"path/filepath"

	"github.com/cloudfoundry/packit"
)

const LayerNameNodeModules = "node_modules"

//go:generate faux --interface PackageManager --output fakes/package_manager.go
type PackageManager interface {
	Install(dir string) error
}

func Build(packageManager PackageManager) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		nodeModulesLayer, err := context.Layers.Get(LayerNameNodeModules, packit.LaunchLayer)
		if err != nil {
			return packit.BuildResult{}, err
		}

		if err = nodeModulesLayer.Reset(); err != nil {
			return packit.BuildResult{}, err
		}

		err = os.Mkdir(filepath.Join(nodeModulesLayer.Path, "node_modules"), os.ModePerm)
		if err != nil {
			return packit.BuildResult{}, err
		}

		err = os.Symlink(filepath.Join(nodeModulesLayer.Path, "node_modules"), filepath.Join(context.WorkingDir, "node_modules"))
		if err != nil {
			return packit.BuildResult{}, err
		}

		err = packageManager.Install(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		return packit.BuildResult{
			Plan:   context.Plan,
			Layers: []packit.Layer{nodeModulesLayer},
			Processes: []packit.Process{
				{
					Type:    "web",
					Command: "npm start",
				},
			},
		}, nil
	}
}
