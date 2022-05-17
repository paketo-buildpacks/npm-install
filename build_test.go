package npminstall_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/paketo-buildpacks/npm-install/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"

	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		workingDir string
		cnbDir     string

		processLayerDir   string
		processWorkingDir string
		processCacheDir   string
		processNpmrcPath  string

		projectPathParser    *fakes.PathParser
		buildProcess         *fakes.BuildProcess
		buildManager         *fakes.BuildManager
		configurationManager *fakes.ConfigurationManager
		entryResolver        *fakes.EntryResolver
		pruneProcess         *fakes.PruneProcess
		sbomGenerator        *fakes.SBOMGenerator
		build                packit.BuildFunc

		buffer *bytes.Buffer
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		projectPathParser = &fakes.PathParser{}
		projectPathParser.GetCall.Returns.ProjectPath = ""

		buildProcess = &fakes.BuildProcess{}
		buildProcess.ShouldRunCall.Returns.Run = true
		buildProcess.ShouldRunCall.Returns.Sha = "some-sha"
		buildProcess.RunCall.Stub = func(ld, cd, wd, rc string, l bool) error {
			err := os.MkdirAll(filepath.Join(ld, "node_modules"), os.ModePerm)
			if err != nil {
				return err
			}

			err = os.MkdirAll(filepath.Join(cd, "layer-content"), os.ModePerm)
			if err != nil {
				return err
			}
			processLayerDir = ld
			processCacheDir = cd
			processWorkingDir = wd
			processNpmrcPath = rc

			return nil
		}

		buildManager = &fakes.BuildManager{}
		buildManager.ResolveCall.Returns.BuildProcess = buildProcess

		configurationManager = &fakes.ConfigurationManager{}

		entryResolver = &fakes.EntryResolver{}

		buffer = bytes.NewBuffer(nil)
		logger := scribe.NewEmitter(buffer)

		pruneProcess = &fakes.PruneProcess{}

		sbomGenerator = &fakes.SBOMGenerator{}
		sbomGenerator.GenerateCall.Returns.SBOM = sbom.SBOM{}

		build = npminstall.Build(
			projectPathParser,
			entryResolver,
			configurationManager,
			buildManager,
			pruneProcess,
			chronos.DefaultClock,
			logger,
			sbomGenerator,
		)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("when required during build", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Build = true
		})
		it("returns a result that installs build modules", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					SBOMFormats: []string{"application/vnd.cyclonedx+json", "application/spdx+json", "application/vnd.syft+json"},
				},
				Platform: packit.Platform{
					Path: "some-platform-path",
				},
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Layers:     packit.Layers{Path: layersDir},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(len(result.Layers)).To(Equal(2))

			buildLayer := result.Layers[0]
			Expect(buildLayer.Name).To(Equal("build-modules"))
			Expect(buildLayer.Path).To(Equal(filepath.Join(layersDir, "build-modules")))
			Expect(buildLayer.SharedEnv).To(Equal(packit.Environment{}))
			Expect(buildLayer.BuildEnv).To(Equal(packit.Environment{
				"PATH.append":       filepath.Join(layersDir, "build-modules", "node_modules", ".bin"),
				"PATH.delim":        ":",
				"NODE_ENV.override": "development",
			}))
			Expect(buildLayer.LaunchEnv).To(Equal(packit.Environment{}))
			Expect(buildLayer.ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(buildLayer.Build).To(BeTrue())
			Expect(buildLayer.Launch).To(BeFalse())
			Expect(buildLayer.Cache).To(BeTrue())
			Expect(buildLayer.Metadata).To(Equal(
				map[string]interface{}{
					"cache_sha": "some-sha",
				}))
			Expect(buildLayer.SBOM.Formats()).To(Equal([]packit.SBOMFormat{
				{
					Extension: "cdx.json",
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.CycloneDXFormat),
				},
				{
					Extension: "spdx.json",
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SPDXFormat),
				},
				{
					Extension: "syft.json",
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SyftFormat),
				},
			}))

			cacheLayer := result.Layers[1]
			Expect(cacheLayer.Name).To(Equal(npminstall.LayerNameCache))
			Expect(cacheLayer.Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
			Expect(cacheLayer.SharedEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.BuildEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.LaunchEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(cacheLayer.Build).To(BeFalse())
			Expect(cacheLayer.Launch).To(BeFalse())
			Expect(cacheLayer.Cache).To(BeTrue())

			Expect(projectPathParser.GetCall.Receives.Path).To(Equal(workingDir))

			Expect(configurationManager.DeterminePathCall.Receives.Typ).To(Equal("npmrc"))
			Expect(configurationManager.DeterminePathCall.Receives.PlatformDir).To(Equal("some-platform-path"))
			Expect(configurationManager.DeterminePathCall.Receives.Entry).To(Equal(".npmrc"))

			Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(workingDir))

			Expect(processLayerDir).To(Equal(filepath.Join(layersDir, "build-modules")))
			Expect(processCacheDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
			Expect(processWorkingDir).To(Equal(workingDir))
			Expect(processNpmrcPath).To(Equal(""))

			workingDirInfo, err := os.Stat(workingDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(workingDirInfo.Mode()).To(Equal(os.FileMode(os.ModeDir | 0775)))

		})
	})

	context("when required during launch", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
		})
		it("returns a result that installs build modules", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					SBOMFormats: []string{"application/vnd.cyclonedx+json", "application/spdx+json", "application/vnd.syft+json"},
				},
				Platform: packit.Platform{
					Path: "some-platform-path",
				},
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(len(result.Layers)).To(Equal(2))

			launchLayer := result.Layers[0]
			Expect(launchLayer.Name).To(Equal("launch-modules"))
			Expect(launchLayer.Path).To(Equal(filepath.Join(layersDir, "launch-modules")))
			Expect(launchLayer.SharedEnv).To(Equal(packit.Environment{}))
			Expect(launchLayer.BuildEnv).To(Equal(packit.Environment{}))
			Expect(launchLayer.LaunchEnv).To(Equal(packit.Environment{
				"NPM_CONFIG_LOGLEVEL.default": "error",
				"PATH.append":                 filepath.Join(layersDir, "launch-modules", "node_modules", ".bin"),
				"PATH.delim":                  ":",
			}))
			Expect(launchLayer.ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(launchLayer.Build).To(BeFalse())
			Expect(launchLayer.Launch).To(BeTrue())
			Expect(launchLayer.Cache).To(BeFalse())
			Expect(launchLayer.Metadata).To(Equal(
				map[string]interface{}{
					"cache_sha": "some-sha",
				}))
			Expect(launchLayer.ExecD).To(Equal([]string{
				filepath.Join(cnbDir, "bin", "setup-symlinks"),
			}))

			Expect(launchLayer.SBOM.Formats()).To(Equal([]packit.SBOMFormat{
				{
					Extension: "cdx.json",
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.CycloneDXFormat),
				},
				{
					Extension: "spdx.json",
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SPDXFormat),
				},
				{
					Extension: "syft.json",
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SyftFormat),
				},
			}))

			cacheLayer := result.Layers[1]
			Expect(cacheLayer.Name).To(Equal(npminstall.LayerNameCache))
			Expect(cacheLayer.Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
			Expect(cacheLayer.SharedEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.BuildEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.LaunchEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(cacheLayer.Build).To(BeFalse())
			Expect(cacheLayer.Launch).To(BeFalse())
			Expect(cacheLayer.Cache).To(BeTrue())

			Expect(projectPathParser.GetCall.Receives.Path).To(Equal(workingDir))

			Expect(configurationManager.DeterminePathCall.Receives.Typ).To(Equal("npmrc"))
			Expect(configurationManager.DeterminePathCall.Receives.PlatformDir).To(Equal("some-platform-path"))
			Expect(configurationManager.DeterminePathCall.Receives.Entry).To(Equal(".npmrc"))

			Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(workingDir))

			Expect(pruneProcess.RunCall.CallCount).To(Equal(0))

			Expect(processLayerDir).To(Equal(filepath.Join(layersDir, "launch-modules")))
			Expect(processCacheDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
			Expect(processWorkingDir).To(Equal(workingDir))
			Expect(processNpmrcPath).To(Equal(""))
		})
	})

	context("when node_modules is required at build and launch", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			entryResolver.MergeLayerTypesCall.Returns.Build = true
		})
		it("resolves and calls the build process", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					SBOMFormats: []string{"application/vnd.cyclonedx+json", "application/spdx+json", "application/vnd.syft+json"},
				},
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "node_modules",
							Metadata: map[string]interface{}{
								"build":  true,
								"launch": true,
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(len(result.Layers)).To(Equal(3))

			buildLayer := result.Layers[0]
			Expect(buildLayer.Name).To(Equal("build-modules"))
			Expect(buildLayer.Path).To(Equal(filepath.Join(layersDir, "build-modules")))
			Expect(buildLayer.SharedEnv).To(Equal(packit.Environment{}))
			Expect(buildLayer.BuildEnv).To(Equal(packit.Environment{
				"PATH.append":       filepath.Join(layersDir, "build-modules", "node_modules", ".bin"),
				"PATH.delim":        ":",
				"NODE_ENV.override": "development",
			}))
			Expect(buildLayer.LaunchEnv).To(Equal(packit.Environment{}))
			Expect(buildLayer.ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(buildLayer.Build).To(BeTrue())
			Expect(buildLayer.Launch).To(BeFalse())
			Expect(buildLayer.Cache).To(BeTrue())
			Expect(buildLayer.Metadata).To(Equal(
				map[string]interface{}{
					"cache_sha": "some-sha",
				}))
			Expect(buildLayer.SBOM.Formats()).To(Equal([]packit.SBOMFormat{
				{
					Extension: "cdx.json",
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.CycloneDXFormat),
				},
				{
					Extension: "spdx.json",
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SPDXFormat),
				},
				{
					Extension: "syft.json",
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SyftFormat),
				},
			}))

			launchLayer := result.Layers[1]
			Expect(launchLayer.Name).To(Equal("launch-modules"))
			Expect(launchLayer.Path).To(Equal(filepath.Join(layersDir, "launch-modules")))
			Expect(launchLayer.SharedEnv).To(Equal(packit.Environment{}))
			Expect(launchLayer.BuildEnv).To(Equal(packit.Environment{}))
			Expect(launchLayer.LaunchEnv).To(Equal(packit.Environment{
				"NPM_CONFIG_LOGLEVEL.default": "error",
				"PATH.append":                 filepath.Join(layersDir, "launch-modules", "node_modules", ".bin"),
				"PATH.delim":                  ":",
			}))
			Expect(launchLayer.ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(launchLayer.Build).To(BeFalse())
			Expect(launchLayer.Launch).To(BeTrue())
			Expect(launchLayer.Cache).To(BeFalse())
			Expect(launchLayer.Metadata).To(Equal(
				map[string]interface{}{
					"cache_sha": "some-sha",
				}))

			Expect(launchLayer.SBOM.Formats()).To(Equal([]packit.SBOMFormat{
				{
					Extension: "cdx.json",
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.CycloneDXFormat),
				},
				{
					Extension: "spdx.json",
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SPDXFormat),
				},
				{
					Extension: "syft.json",
					Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SyftFormat),
				},
			}))

			cacheLayer := result.Layers[2]
			Expect(cacheLayer.Name).To(Equal(npminstall.LayerNameCache))
			Expect(cacheLayer.Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
			Expect(cacheLayer.SharedEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.BuildEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.LaunchEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(cacheLayer.Build).To(BeFalse())
			Expect(cacheLayer.Launch).To(BeFalse())
			Expect(cacheLayer.Cache).To(BeTrue())

			Expect(pruneProcess.RunCall.Receives.ModulesDir).To(Equal(launchLayer.Path))
			Expect(pruneProcess.RunCall.Receives.CacheDir).To(Equal(cacheLayer.Path))
			Expect(pruneProcess.RunCall.Receives.WorkingDir).To(Equal(workingDir))
			Expect(pruneProcess.RunCall.Receives.NpmrcPath).To(Equal(""))
		})
	})

	context("when one npmrc binding is detected", func() {
		it.Before(func() {
			configurationManager.DeterminePathCall.Returns.Path = "some-binding-path/.npmrc"
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
		})

		it("passes the path to the .npmrc to the build process and env configuration", func() {
			_, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(buildProcess.ShouldRunCall.Receives.NpmrcPath).To(Equal("some-binding-path/.npmrc"))
			Expect(buildProcess.RunCall.Receives.NpmrcPath).To(Equal("some-binding-path/.npmrc"))
		})

		context("and NPM_CONFIG_GLOBALCONFIG is already set in the build environment", func() {
			it.Before(func() {
				os.Setenv("NPM_CONFIG_GLOBALCONFIG", "some/path/.npmrc")
			})
			it.After(func() {
				os.Unsetenv("NPM_CONFIG_GLOBALCONFIG")
			})

			it("does not change the previously set value of the env var", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(buildProcess.ShouldRunCall.Receives.NpmrcPath).To(Equal("some/path/.npmrc"))
				Expect(buildProcess.RunCall.Receives.NpmrcPath).To(Equal("some/path/.npmrc"))
			})
		})
	})

	context("when the build process should not run", func() {
		it.Before(func() {
			buildProcess.ShouldRunCall.Returns.Run = false
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm))
		})

		it("resolves and skips build process", func() {
			_, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(projectPathParser.GetCall.Receives.Path).To(Equal(workingDir))
			Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(workingDir))
			Expect(buildProcess.RunCall.CallCount).To(Equal(0))

			link, err := os.Readlink(filepath.Join(workingDir, "node_modules"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layersDir, "launch-modules", "node_modules")))
		})

		context("when BP_NODE_PROJECT_PATH is set", func() {
			it.Before(func() {
				buildProcess.ShouldRunCall.Returns.Run = true
				projectPathParser.GetCall.Returns.ProjectPath = "some-dir"
				Expect(os.MkdirAll(filepath.Join(workingDir, "some-dir", "node_modules"), os.ModePerm))
			})

			it("resolves and calls the build process", func() {
				_, err := build(packit.BuildContext{
					BuildpackInfo: packit.BuildpackInfo{
						SBOMFormats: []string{"application/vnd.cyclonedx+json", "application/spdx+json", "application/vnd.syft+json"},
					},
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(projectPathParser.GetCall.Receives.Path).To(Equal(workingDir))

				Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(filepath.Join(workingDir, "some-dir")))

				Expect(processLayerDir).To(Equal(filepath.Join(layersDir, "launch-modules")))
				Expect(processCacheDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
				Expect(processWorkingDir).To(Equal(filepath.Join(workingDir, "some-dir")))

				procWorkingDirInfo, err := os.Stat(processWorkingDir)
				Expect(err).NotTo(HaveOccurred())
				Expect(procWorkingDirInfo.Mode()).To(Equal(os.FileMode(os.ModeDir | 0775)))
			})
		})

	})

	context("when the cache layer directory is empty", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			buildProcess.RunCall.Stub = func(ld, cd, wd, rc string, l bool) error {
				err := os.MkdirAll(cd, os.ModePerm)
				if err != nil {
					return err
				}

				return nil
			}
		})

		it("filters out empty layers", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(len(result.Layers)).To(Equal(1))
			Expect(result.Layers[0].Name).To(Equal("launch-modules"))
		})
	})

	context("when the cache layer directory does not exist", func() {
		it("filters out empty layers", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(len(result.Layers)).To(Equal(0))
		})
	})

	context("failure cases", func() {
		context("when the npm-cache layer cannot be fetched", func() {
			it.Before(func() {
				_, err := os.Create(filepath.Join(layersDir, "npm-cache.toml"))
				Expect(err).NotTo(HaveOccurred())
				Expect(os.Chmod(filepath.Join(layersDir, "npm-cache.toml"), 0000)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the configuration manager fails while determining the path", func() {
			it.Before(func() {
				configurationManager.DeterminePathCall.Returns.Err = errors.New("failed to determine path")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).To(MatchError("failed to determine path"))
			})
		})

		context("when the project path parser provided fails", func() {
			it.Before(func() {
				projectPathParser.GetCall.Returns.Err = errors.New("failed to parse project path")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).To(MatchError("failed to parse project path"))
			})
		})

		context("when the build process cannot be resolved", func() {
			it.Before(func() {
				buildManager.ResolveCall.Returns.Error = errors.New("failed to resolve build process")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).To(MatchError("failed to resolve build process"))
			})
		})

		context("during the build installation process", func() {
			it.Before(func() {
				entryResolver.MergeLayerTypesCall.Returns.Build = true
			})

			context("when the node_modules layer cannot be fetched", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(layersDir, "build-modules.toml"), nil, 0000)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the build process cache check fails", func() {
				it.Before(func() {
					buildProcess.ShouldRunCall.Returns.Err = errors.New("failed to check cache")
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError("failed to check cache"))
				})
			})

			context("when the node_modules layer cannot be reset", func() {
				it.Before(func() {
					Expect(os.Chmod(layersDir, 4444)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(layersDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the build process provided fails", func() {
				it.Before(func() {
					buildProcess.RunCall.Stub = func(string, string, string, string, bool) error {
						return errors.New("given build process failed")
					}
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError("given build process failed"))
				})
			})

			context("when the BOM cannot be generated", func() {
				it.Before(func() {
					sbomGenerator.GenerateCall.Returns.Error = errors.New("failed to generate SBOM")
				})
				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						BuildpackInfo: packit.BuildpackInfo{
							SBOMFormats: []string{"application/vnd.cyclonedx+json", "application/spdx+json", "application/vnd.syft+json"},
						},
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{{Name: "node_modules"}},
						},
						Stack: "some-stack",
					})
					Expect(err).To(MatchError("failed to generate SBOM"))
				})
			})

			context("when the BOM cannot be formatted", func() {
				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						Layers:  packit.Layers{Path: layersDir},
						CNBPath: cnbDir,
						BuildpackInfo: packit.BuildpackInfo{
							SBOMFormats: []string{"random-format"},
						},
					})
					Expect(err).To(MatchError("unsupported SBOM format: 'random-format'"))
				})
			})

			context("when BP_DISABLE_SBOM is set incorrectly", func() {
				it.Before((func() {
					os.Setenv("BP_DISABLE_SBOM", "not-a-bool")
				}))

				it.After((func() {
					os.Unsetenv("BP_DISABLE_SBOM")
				}))

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						Layers:  packit.Layers{Path: layersDir},
						CNBPath: cnbDir,
						BuildpackInfo: packit.BuildpackInfo{
							SBOMFormats: []string{"application/vnd.cyclonedx+json"},
						},
					})
					Expect(err).To(MatchError(ContainSubstring("failed to parse BP_DISABLE_SBOM")))
				})
			})

			context("when the node_modules directory cannot be removed", func() {
				it.Before(func() {
					buildProcess.ShouldRunCall.Returns.Run = false
					Expect(os.Chmod(workingDir, 0000))
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm))
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})

		context("during the launch installation process", func() {
			it.Before(func() {
				entryResolver.MergeLayerTypesCall.Returns.Launch = true
			})

			context("when the node_modules layer cannot be fetched", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(layersDir, "launch-modules.toml"), nil, 0000)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the build process cache check fails", func() {
				it.Before(func() {
					buildProcess.ShouldRunCall.Returns.Err = errors.New("failed to check cache")
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError("failed to check cache"))
				})
			})

			context("when the node_modules layer cannot be reset", func() {
				it.Before(func() {
					Expect(os.Chmod(layersDir, 4444)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(layersDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when build is also set and the node_modules copy fails", func() {
				it.Before(func() {
					entryResolver.MergeLayerTypesCall.Returns.Build = true
					buildProcess.RunCall.Stub = func(string, string, string, string, bool) error {
						return nil
					}
				})
				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
				})
			})

			context("when the build process provided fails", func() {
				it.Before(func() {
					buildProcess.RunCall.Stub = func(string, string, string, string, bool) error {
						return errors.New("given build process failed")
					}
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError("given build process failed"))
				})
			})

			context("when the build process provided fails", func() {
				context("when build is also set", func() {
					it.Before(func() {
						entryResolver.MergeLayerTypesCall.Returns.Build = true
						pruneProcess.RunCall.Returns.Error = errors.New("prune process failed")
					})

					it("returns an error", func() {
						_, err := build(packit.BuildContext{
							WorkingDir: workingDir,
							Layers:     packit.Layers{Path: layersDir},
							CNBPath:    cnbDir,
							Plan: packit.BuildpackPlan{
								Entries: []packit.BuildpackPlanEntry{
									{Name: "node_modules"},
								},
							},
						})
						Expect(err).To(MatchError("prune process failed"))
					})
				})
			})

			context("when the BOM cannot be generated", func() {
				it.Before(func() {
					sbomGenerator.GenerateCall.Returns.Error = errors.New("failed to generate SBOM")
				})
				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						BuildpackInfo: packit.BuildpackInfo{
							SBOMFormats: []string{"application/vnd.cyclonedx+json", "application/spdx+json", "application/vnd.syft+json"},
						},
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{{Name: "node_modules"}},
						},
						Stack: "some-stack",
					})
					Expect(err).To(MatchError("failed to generate SBOM"))
				})
			})

			context("when the BOM cannot be formatted", func() {
				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						Layers:  packit.Layers{Path: layersDir},
						CNBPath: cnbDir,
						BuildpackInfo: packit.BuildpackInfo{
							SBOMFormats: []string{"random-format"},
						},
					})
					Expect(err).To(MatchError("unsupported SBOM format: 'random-format'"))
				})
			})

			context("when BP_DISABLE_SBOM is set incorrectly", func() {
				it.Before((func() {
					os.Setenv("BP_DISABLE_SBOM", "not-a-bool")
				}))

				it.After((func() {
					os.Unsetenv("BP_DISABLE_SBOM")
				}))

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						Layers:  packit.Layers{Path: layersDir},
						CNBPath: cnbDir,
						BuildpackInfo: packit.BuildpackInfo{
							SBOMFormats: []string{"application/vnd.cyclonedx+json"},
						},
					})
					Expect(err).To(MatchError(ContainSubstring("failed to parse BP_DISABLE_SBOM")))
				})
			})

			context("when the node_modules directory cannot be removed", func() {
				it.Before(func() {
					buildProcess.ShouldRunCall.Returns.Run = false
					Expect(os.Chmod(workingDir, 0000))
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm))
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
