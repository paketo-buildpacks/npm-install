package npminstall

import (
	"os"
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit/v2/sbom"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface BuildManager --output fakes/build_manager.go
type BuildManager interface {
	Resolve(workingDir, cacheDir string) (BuildProcess, error)
}

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	MergeLayerTypes(string, []packit.BuildpackPlanEntry) (launch, build bool)
}

//go:generate faux --interface SBOMGenerator --output fakes/sbom_generator.go
type SBOMGenerator interface {
	Generate(dir string) (sbom.SBOM, error)
}

//go:generate faux --interface ConfigurationManager --output fakes/configuration_manager.go
type ConfigurationManager interface {
	DeterminePath(typ, platformDir, entry string) (path string, err error)
}

//go:generate faux --interface PruneProcess --output fakes/prune_process.go
type PruneProcess interface {
	ShouldRun(workingDir string, metadata map[string]interface{}, npmrcPath string) (run bool, sha string, err error)
	Run(modulesDir, cacheDir, workingDir, npmrcPath string, launch bool) error
}

func Build(projectPathParser PathParser,
	entryResolver EntryResolver,
	configurationManager ConfigurationManager,
	buildManager BuildManager,
	pruneProcess PruneProcess,
	clock chronos.Clock,
	logger scribe.Emitter,
	sbomGenerator SBOMGenerator) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)

		launch, build := entryResolver.MergeLayerTypes(NodeModules, context.Plan.Entries)

		npmCacheLayer, err := context.Layers.Get(LayerNameCache)
		if err != nil {
			return packit.BuildResult{}, err
		}

		npmCacheLayer.Cache = true

		var globalNpmrcPath string
		path, ok := os.LookupEnv("NPM_CONFIG_GLOBALCONFIG")
		if ok {
			globalNpmrcPath = path
		} else {
			var err error
			globalNpmrcPath, err = configurationManager.DeterminePath("npmrc", context.Platform.Path, ".npmrc")
			if err != nil {
				return packit.BuildResult{}, err
			}
		}

		logger.Process("Resolving installation process")

		projectPath, err := projectPathParser.Get(context.WorkingDir)
		if err != nil {
			return packit.BuildResult{}, err
		}

		projectPath = filepath.Join(context.WorkingDir, projectPath)

		process, err := buildManager.Resolve(projectPath, npmCacheLayer.Path)
		if err != nil {
			return packit.BuildResult{}, err
		}

		var layers []packit.Layer
		var buildLayerPath string
		if build {
			layer, err := context.Layers.Get("build-modules")
			if err != nil {
				return packit.BuildResult{}, err
			}
			buildLayerPath = layer.Path

			run, sha, err := process.ShouldRun(projectPath, layer.Metadata, globalNpmrcPath)
			if err != nil {
				return packit.BuildResult{}, err
			}

			if run {
				logger.Process("Executing build environment install process")

				layer, err = layer.Reset()
				if err != nil {
					return packit.BuildResult{}, err
				}

				duration, err := clock.Measure(func() error {
					return process.Run(layer.Path, npmCacheLayer.Path, projectPath, globalNpmrcPath, false)
				})
				if err != nil {
					return packit.BuildResult{}, err
				}

				logger.Action("Completed in %s", duration.Round(time.Millisecond))
				logger.Break()

				layer.Metadata = map[string]interface{}{
					"cache_sha": sha,
				}

				if globalNpmrcPath != "" {
					layer.BuildEnv.Default("NPM_CONFIG_GLOBALCONFIG", globalNpmrcPath)
				}
				path := filepath.Join(layer.Path, "node_modules", ".bin")
				layer.BuildEnv.Append("PATH", path, string(os.PathListSeparator))
				layer.BuildEnv.Override("NODE_ENV", "development")

				logger.EnvironmentVariables(layer)

				logger.GeneratingSBOM(layer.Path)

				var sbomContent sbom.SBOM
				duration, err = clock.Measure(func() error {
					sbomContent, err = sbomGenerator.Generate(context.WorkingDir)
					return err
				})
				if err != nil {
					return packit.BuildResult{}, err
				}
				logger.Action("Completed in %s", duration.Round(time.Millisecond))
				logger.Break()

				logger.FormattingSBOM(context.BuildpackInfo.SBOMFormats...)

				layer.SBOM, err = sbomContent.InFormats(context.BuildpackInfo.SBOMFormats...)
				if err != nil {
					return packit.BuildResult{}, err
				}
			} else {
				logger.Process("Reusing cached layer %s", layer.Path)
				err := os.RemoveAll(filepath.Join(projectPath, "node_modules"))
				if err != nil {
					return packit.BuildResult{}, err
				}

				err = os.Symlink(filepath.Join(layer.Path, "node_modules"), filepath.Join(projectPath, "node_modules"))
				if err != nil {
					return packit.BuildResult{}, err
				}
			}
			layer.Build = true
			layer.Cache = true

			layers = append(layers, layer)
		}

		if launch {
			layer, err := context.Layers.Get("launch-modules")
			if err != nil {
				return packit.BuildResult{}, err
			}

			run, sha, err := process.ShouldRun(projectPath, layer.Metadata, globalNpmrcPath)
			if err != nil {
				return packit.BuildResult{}, err
			}

			if run {
				logger.Process("Executing launch environment install process")

				layer, err = layer.Reset()
				if err != nil {
					return packit.BuildResult{}, err
				}

				if build {
					err := fs.Copy(filepath.Join(buildLayerPath, "node_modules"), filepath.Join(projectPath, "node_modules"))
					if err != nil {
						return packit.BuildResult{}, err
					}
					process = pruneProcess
				}

				duration, err := clock.Measure(func() error {
					return process.Run(layer.Path, npmCacheLayer.Path, projectPath, globalNpmrcPath, true)
				})
				if err != nil {
					return packit.BuildResult{}, err
				}

				if build {
					err = fs.Move(filepath.Join(projectPath, "node_modules"), filepath.Join(layer.Path, "node_modules"))
					if err != nil {
						return packit.BuildResult{}, err
					}

					err = os.Symlink(filepath.Join(buildLayerPath, "node_modules"), filepath.Join(projectPath, "node_modules"))
					if err != nil {
						return packit.BuildResult{}, err
					}

				}

				logger.Action("Completed in %s", duration.Round(time.Millisecond))
				logger.Break()

				layer.Metadata = map[string]interface{}{
					"cache_sha": sha,
				}

				layer.LaunchEnv.Default("NPM_CONFIG_LOGLEVEL", "error")
				path := filepath.Join(layer.Path, "node_modules", ".bin")
				layer.LaunchEnv.Append("PATH", path, string(os.PathListSeparator))

				logger.EnvironmentVariables(layer)

				logger.GeneratingSBOM(layer.Path)

				var sbomContent sbom.SBOM
				duration, err = clock.Measure(func() error {
					sbomContent, err = sbomGenerator.Generate(context.WorkingDir)
					return err
				})
				if err != nil {
					return packit.BuildResult{}, err
				}
				logger.Action("Completed in %s", duration.Round(time.Millisecond))
				logger.Break()

				logger.FormattingSBOM(context.BuildpackInfo.SBOMFormats...)

				layer.SBOM, err = sbomContent.InFormats(context.BuildpackInfo.SBOMFormats...)
				if err != nil {
					return packit.BuildResult{}, err
				}

				layer.ExecD = []string{filepath.Join(context.CNBPath, "bin", "setup-symlinks")}
			} else {
				logger.Process("Reusing cached layer %s", layer.Path)
				if !build {
					err := os.RemoveAll(filepath.Join(projectPath, "node_modules"))
					if err != nil {
						return packit.BuildResult{}, err
					}

					err = os.Symlink(filepath.Join(layer.Path, "node_modules"), filepath.Join(projectPath, "node_modules"))
					if err != nil {
						return packit.BuildResult{}, err
					}
				}
			}

			layer.Launch = true

			layers = append(layers, layer)
		}

		exists, err := fs.Exists(npmCacheLayer.Path)
		if exists {
			if !fs.IsEmptyDir(npmCacheLayer.Path) {
				layers = append(layers, npmCacheLayer)
			}
		}
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Break()

		return packit.BuildResult{Layers: layers}, nil
	}
}
