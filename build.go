package npminstall

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/packit/v2/servicebindings"
)

//go:generate faux --interface BuildManager --output fakes/build_manager.go
type BuildManager interface {
	Resolve(workingDir, cacheDir string) (BuildProcess, error)
}

//go:generate faux --interface BindingResolver --output fakes/binding_resolver.go
type BindingResolver interface {
	Resolve(typ, provider, platformDir string) ([]servicebindings.Binding, error)
}

//go:generate faux --interface EnvironmentConfig --output fakes/environment_config.go
type EnvironmentConfig interface {
	Configure(layer packit.Layer, npmrcPath string) error
	GetValue(key string) string
}

func Build(projectPathParser PathParser,
	bindingResolver BindingResolver,
	buildManager BuildManager,
	clock chronos.Clock,
	environment EnvironmentConfig,
	logger scribe.Logger) packit.BuildFunc {
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

		var globalNpmrcPath string
		bindings, err := bindingResolver.Resolve("npmrc", "", context.Platform.Path)
		if err != nil {
			return packit.BuildResult{}, err
		}

		if len(bindings) > 1 {
			return packit.BuildResult{}, errors.New("binding resolver found more than one binding of type 'npmrc'")
		}

		if len(bindings) == 1 {
			logger.Process("Loading npmrc service binding")

			npmrcExists := false
			for key := range bindings[0].Entries {
				if key == ".npmrc" {
					npmrcExists = true
					break
				}
			}
			if !npmrcExists {
				return packit.BuildResult{}, errors.New("binding of type 'npmrc' does not contain required entry '.npmrc'")
			}
			globalNpmrcPath = filepath.Join(bindings[0].Path, ".npmrc")
		}

		if path, ok := os.LookupEnv("NPM_CONFIG_GLOBALCONFIG"); ok {
			globalNpmrcPath = path
		}

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

		run, sha, err := process.ShouldRun(projectPath, nodeModulesLayer.Metadata, globalNpmrcPath)
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
				return process.Run(nodeModulesLayer.Path, nodeCacheLayer.Path, projectPath, globalNpmrcPath)
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

			err = environment.Configure(nodeModulesLayer, globalNpmrcPath)
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
