package npm

import (
	"os"
	"path/filepath"
	"time"

	"github.com/cloudfoundry/packit"
	"github.com/cloudfoundry/packit/fs"
	"github.com/cloudfoundry/packit/scribe"
)

const (
	LayerNameNodeModules = "modules"
	LayerNameCache       = "npm-cache"
)

//go:generate faux --interface BuildManager --output fakes/build_manager.go
type BuildManager interface {
	Resolve(workingDir, cacheDir string) (BuildProcess, error)
}

func Build(buildManager BuildManager, clock Clock, logger scribe.Logger) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", "<Buildpack Name>", "<Buildpack Version>")
		logger.Process("Resolving NPM build process")

		nodeModulesLayer, err := context.Layers.Get(LayerNameNodeModules, packit.LaunchLayer)
		if err != nil {
			return packit.BuildResult{}, err
		}

		nodeCacheLayer, err := context.Layers.Get(LayerNameCache, packit.CacheLayer)
		if err != nil {
			return packit.BuildResult{}, err
		}

		process, err := buildManager.Resolve(context.WorkingDir, nodeCacheLayer.Path)
		if err != nil {
			return packit.BuildResult{}, err
		}

		run, sha, err := process.ShouldRun(context.WorkingDir, nodeModulesLayer.Metadata)
		if err != nil {
			return packit.BuildResult{}, err
		}

		if run {
			if err = nodeModulesLayer.Reset(); err != nil {
				return packit.BuildResult{}, err
			}

			err = process.Run(nodeModulesLayer.Path, nodeCacheLayer.Path, context.WorkingDir)
			if err != nil {
				return packit.BuildResult{}, err
			}

			nodeModulesLayer.Metadata = map[string]interface{}{
				"built_at":  clock.Now().Format(time.RFC3339Nano),
				"cache_sha": sha,
			}
		} else {
			err := os.RemoveAll(filepath.Join(context.WorkingDir, "node_modules"))
			if err != nil {
				return packit.BuildResult{}, err
			}

			err = os.Symlink(filepath.Join(nodeModulesLayer.Path, "node_modules"), filepath.Join(context.WorkingDir, "node_modules"))
			if err != nil {
				return packit.BuildResult{}, err
			}
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
