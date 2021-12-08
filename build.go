package npminstall

import (
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"os"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface BuildManager --output fakes/build_manager.go
type BuildManager interface {
	Resolve(workingDir, cacheDir string) (BuildProcess, error)
}

//go:generate faux --interface EnvironmentConfig --output fakes/environment_config.go
type EnvironmentConfig interface {
	Configure(layer packit.Layer) error
	GetValue(key string) string
}

func Build(projectPathParser PathParser, buildManager BuildManager, clock chronos.Clock, environment EnvironmentConfig, logger scribe.Logger) packit.BuildFunc {
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

		projectPath, err := projectPathParser.Get(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		projectPath = filepath.Join(context.WorkingDir, projectPath)

		process, err := buildManager.Resolve(projectPath, nodeCacheLayer.Path)
		if err != nil {
			return packit.BuildResult{}, err
		}

		run, sha, err := process.ShouldRun(projectPath, nodeModulesLayer.Metadata)
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
				return process.Run(nodeModulesLayer.Path, nodeCacheLayer.Path, projectPath)
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

			err = environment.Configure(nodeModulesLayer)
			if err != nil {
				return packit.BuildResult{}, err
			}
		} else {
			logger.Process("Reusing cached layer %s", nodeModulesLayer.Path)
			err := os.RemoveAll(filepath.Join(projectPath, "node_modules"))
			if err != nil {
				return packit.BuildResult{}, err
			}

			err = os.Symlink(filepath.Join(nodeModulesLayer.Path, "node_modules"), filepath.Join(projectPath, "node_modules"))
			if err != nil {
				return packit.BuildResult{}, err
			}
		}

		// run sbom.Generate path to working dir
		// set the sbom.Generate.inFormaters(context.buildinfo.sbom syft, cdx) ["application/vnd.cyclonedx+json", "application/vnd.syft+json"]
		// set the sbomformatter on the layer
		sBom, err := sbom.Generate(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}
		formatter, err := sBom.InFormats(context.BuildpackInfo.SBOMFormats...)
		if err != nil {
			return packit.BuildResult{}, err
		}
		nodeModulesLayer.SBOM = formatter

		layers := []packit.Layer{nodeModulesLayer}

		if _, err := os.Stat(nodeCacheLayer.Path); err == nil {
			if !fs.IsEmptyDir(nodeCacheLayer.Path) {
				layers = append(layers, nodeCacheLayer)
			}
		}

		logger.Break()

		return packit.BuildResult{Layers: layers}, nil
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
