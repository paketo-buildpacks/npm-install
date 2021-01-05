package npminstall

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/scribe"
)

//go:generate faux --interface BuildManager --output fakes/build_manager.go
type BuildManager interface {
	Resolve(workingDir, cacheDir string) (BuildProcess, error)
}

func Build(buildManager BuildManager, clock chronos.Clock, logger scribe.Logger) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		nodeModulesLayer, err := context.Layers.Get(LayerNameNodeModules)
		if err != nil {
			return packit.BuildResult{}, err
		}
		nodeModulesLayer = setLayerFlags(nodeModulesLayer, context.Plan.Entries)

		nodeCacheLayer, err := context.Layers.Get(LayerNameCache)
		if err != nil {
			return packit.BuildResult{}, err
		}
		nodeCacheLayer = setLayerFlags(nodeCacheLayer, context.Plan.Entries)

		logger.Process("Resolving installation process")

		process, err := buildManager.Resolve(context.WorkingDir, nodeCacheLayer.Path)
		if err != nil {
			return packit.BuildResult{}, err
		}

		run, sha, err := process.ShouldRun(context.WorkingDir, nodeModulesLayer.Metadata)
		if err != nil {
			return packit.BuildResult{}, err
		}

		if run {
			logger.Process("Executing build process")

			nodeModulesLayer, err = nodeModulesLayer.Reset()
			if err != nil {
				return packit.BuildResult{}, err
			}
			nodeModulesLayer = setLayerFlags(nodeModulesLayer, context.Plan.Entries)

			duration, err := clock.Measure(func() error {
				return process.Run(nodeModulesLayer.Path, nodeCacheLayer.Path, context.WorkingDir)
			})
			if err != nil {
				return packit.BuildResult{}, err
			}

			logger.Action("Completed in %s", duration.Round(time.Millisecond))
			logger.Break()

			nodeModulesLayer.Metadata = map[string]interface{}{
				"built_at":  clock.Now().Format(time.RFC3339Nano),
				"cache_sha": sha,
			}

			nodeModulesLayer.LaunchEnv.Override("NPM_CONFIG_LOGLEVEL", "error")
			nodeModulesLayer.LaunchEnv.Override("NPM_CONFIG_PRODUCTION", "true")

			path := filepath.Join(nodeModulesLayer.Path, "node_modules", ".bin")
			nodeModulesLayer.SharedEnv.Append("PATH", path, string(os.PathListSeparator))

			logger.Process("Configuring environment")
			logger.Subprocess("%s", scribe.FormattedMap{
				"NPM_CONFIG_LOGLEVEL":   "error",
				"NPM_CONFIG_PRODUCTION": "true",
				"PATH":                  fmt.Sprintf("$PATH:%s", path),
			})
		} else {
			logger.Process("Reusing cached layer %s", nodeModulesLayer.Path)
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

		logger.Break()

		return packit.BuildResult{
			Plan:   context.Plan,
			Layers: layers,
		}, nil
	}
}

func setLayerFlags(layer packit.Layer, entries []packit.BuildpackPlanEntry) packit.Layer {
	for _, entry := range entries {
		launch, ok := entry.Metadata["launch"].(bool)
		if ok && launch {
			layer.Launch = true
			layer.Cache = true
		}

		build, ok := entry.Metadata["build"].(bool)
		if ok && build {
			layer.Build = true
			layer.Cache = true
		}
	}

	return layer
}
