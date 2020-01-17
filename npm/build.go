package npm

import (
	"os"

	"github.com/cloudfoundry/packit"
	"github.com/cloudfoundry/packit/fs"
)

const (
	LayerNameNodeModules = "modules_layer"
	LayerNameCache       = "npm-cache"
)

//go:generate faux --interface BuildManager --output fakes/build_manager.go
type BuildManager interface {
	Resolve(workingDir, cacheDir string) (BuildProcess, error)
}

func Build(buildManager BuildManager) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {

		nodeModulesLayer, err := context.Layers.Get(LayerNameNodeModules, packit.LaunchLayer)
		if err != nil {
			return packit.BuildResult{}, err
		}

		nodeCacheLayer, err := context.Layers.Get(LayerNameCache, packit.CacheLayer)
		if err != nil {
			return packit.BuildResult{}, err
		}

		if err = nodeModulesLayer.Reset(); err != nil {
			return packit.BuildResult{}, err
		}

		process, err := buildManager.Resolve(context.WorkingDir, nodeCacheLayer.Path)
		if err != nil {
			return packit.BuildResult{}, err
		}

		err = process.Run(nodeModulesLayer.Path, nodeCacheLayer.Path, context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		layers := []packit.Layer{nodeModulesLayer}

		if _, err := os.Stat(nodeCacheLayer.Path); err == nil {
			if !fs.IsEmptyDir(nodeCacheLayer.Path) {
				layers = append(layers, nodeCacheLayer)
			}
		}

		return packit.BuildResult{
			Plan:   context.Plan,
			Layers: layers,
			Processes: []packit.Process{
				{
					Type:    "web",
					Command: "npm start",
				},
			},
		}, nil
	}
}
