package npm

import "github.com/cloudfoundry/packit"

const LayerNameNodeModules = "modules_layer"

//go:generate faux --interface BuildManager --output fakes/build_manager.go
type BuildManager interface {
	Resolve(workingDir string) (BuildProcess, error)
}

func Build(buildManager BuildManager) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		nodeModulesLayer, err := context.Layers.Get(LayerNameNodeModules, packit.LaunchLayer)
		if err != nil {
			return packit.BuildResult{}, err
		}

		if err = nodeModulesLayer.Reset(); err != nil {
			return packit.BuildResult{}, err
		}

		process, err := buildManager.Resolve(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		err = process(nodeModulesLayer.Path, context.WorkingDir)
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
