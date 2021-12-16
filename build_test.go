package npminstall_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

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

		processLayerDir   string
		processWorkingDir string
		processCacheDir   string

		timestamp string

		projectPathParser *fakes.PathParser
		buildProcess      *fakes.BuildProcess
		buildManager      *fakes.BuildManager
		environment       *fakes.EnvironmentConfig
		sbomGenerator     *fakes.SBOMGenerator
		clock             chronos.Clock
		build             packit.BuildFunc

		buffer *bytes.Buffer
	)

	it.Before(func() {
		var err error
		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		projectPathParser = &fakes.PathParser{}
		projectPathParser.GetCall.Returns.ProjectPath = ""

		buildProcess = &fakes.BuildProcess{}
		buildProcess.ShouldRunCall.Returns.Run = true
		buildProcess.ShouldRunCall.Returns.Sha = "some-sha"
		buildProcess.RunCall.Stub = func(ld, cd, wd string) error {
			err := os.MkdirAll(filepath.Join(ld, "layer-content"), os.ModePerm)
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

			return nil
		}

		now := time.Now()
		clock = chronos.NewClock(func() time.Time {
			return now
		})
		timestamp = now.Format(time.RFC3339Nano)

		buildManager = &fakes.BuildManager{}
		buildManager.ResolveCall.Returns.BuildProcess = buildProcess

		environment = &fakes.EnvironmentConfig{}

		buffer = bytes.NewBuffer(nil)
		logger := scribe.NewLogger(buffer)

		sbomGenerator = &fakes.SBOMGenerator{}
		sbomGenerator.GenerateCall.Returns.SBOM = sbom.SBOM{}

		build = npminstall.Build(projectPathParser, buildManager, clock, environment, logger, sbomGenerator)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("resolves and calls the build process", func() {
		result, err := build(packit.BuildContext{
			BuildpackInfo: packit.BuildpackInfo{
				SBOMFormats: []string{"application/vnd.cyclonedx+json", "application/spdx+json", "application/vnd.syft+json"},
			},
			WorkingDir: workingDir,
			Layers:     packit.Layers{Path: layersDir},
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{Name: "node_modules"},
				},
			},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(result.Layers[0].Name).To(Equal(npminstall.LayerNameNodeModules))
		Expect(result.Layers[0].Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameNodeModules)))
		Expect(result.Layers[0].SharedEnv).To(Equal(packit.Environment{}))
		Expect(result.Layers[0].BuildEnv).To(Equal(packit.Environment{}))
		Expect(result.Layers[0].LaunchEnv).To(Equal(packit.Environment{}))
		Expect(result.Layers[0].ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
		Expect(result.Layers[0].Build).To(BeFalse())
		Expect(result.Layers[0].Launch).To(BeFalse())
		Expect(result.Layers[0].Cache).To(BeFalse())
		Expect(result.Layers[0].Metadata).To(Equal(
			map[string]interface{}{
				"built_at":  timestamp,
				"cache_sha": "some-sha",
			}))
		Expect(result.Layers[1].Name).To(Equal(npminstall.LayerNameCache))
		Expect(result.Layers[1].Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
		Expect(result.Layers[1].SharedEnv).To(Equal(packit.Environment{}))
		Expect(result.Layers[1].BuildEnv).To(Equal(packit.Environment{}))
		Expect(result.Layers[1].LaunchEnv).To(Equal(packit.Environment{}))
		Expect(result.Layers[1].ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
		Expect(result.Layers[1].Build).To(BeFalse())
		Expect(result.Layers[1].Launch).To(BeFalse())
		Expect(result.Layers[1].Cache).To(BeFalse())

		Expect(result.Layers[0].SBOM.Formats()).To(Equal([]packit.SBOMFormat{
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

		Expect(projectPathParser.GetCall.Receives.Path).To(Equal(workingDir))
		Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(workingDir))
		Expect(environment.ConfigureCall.CallCount).To(Equal(1))
		Expect(environment.ConfigureCall.Receives.Layer.Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameNodeModules)))

		Expect(processLayerDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameNodeModules)))
		Expect(processCacheDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
		Expect(processWorkingDir).To(Equal(workingDir))
	})

	context("when node_modules is required at build and launch", func() {
		it("resolves and calls the build process", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					SBOMFormats: []string{"application/vnd.cyclonedx+json", "application/spdx+json", "application/vnd.syft+json"},
				},
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
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
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Layers[0].Name).To(Equal(npminstall.LayerNameNodeModules))
			Expect(result.Layers[0].Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameNodeModules)))
			Expect(result.Layers[0].SharedEnv).To(Equal(packit.Environment{}))
			Expect(result.Layers[0].BuildEnv).To(Equal(packit.Environment{}))
			Expect(result.Layers[0].LaunchEnv).To(Equal(packit.Environment{}))
			Expect(result.Layers[0].ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(result.Layers[0].Build).To(BeTrue())
			Expect(result.Layers[0].Launch).To(BeTrue())
			Expect(result.Layers[0].Cache).To(BeTrue())
			Expect(result.Layers[0].Metadata).To(Equal(
				map[string]interface{}{
					"built_at":  timestamp,
					"cache_sha": "some-sha",
				}))
			Expect(result.Layers[1].Name).To(Equal(npminstall.LayerNameCache))
			Expect(result.Layers[1].Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
			Expect(result.Layers[1].SharedEnv).To(Equal(packit.Environment{}))
			Expect(result.Layers[1].BuildEnv).To(Equal(packit.Environment{}))
			Expect(result.Layers[1].LaunchEnv).To(Equal(packit.Environment{}))
			Expect(result.Layers[1].ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(result.Layers[1].Build).To(BeTrue())
			Expect(result.Layers[1].Launch).To(BeTrue())
			Expect(result.Layers[1].Cache).To(BeTrue())

			Expect(result.Layers[0].SBOM.Formats()).To(Equal([]packit.SBOMFormat{
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

			Expect(projectPathParser.GetCall.Receives.Path).To(Equal(workingDir))
			Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(workingDir))

			Expect(processLayerDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameNodeModules)))
			Expect(processCacheDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
			Expect(processWorkingDir).To(Equal(workingDir))
		})
	})

	context("when the build process should not run", func() {
		it.Before(func() {
			buildProcess.ShouldRunCall.Returns.Run = false
			Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm))
		})

		it("resolves and skips build process", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name:             npminstall.LayerNameNodeModules,
						Path:             filepath.Join(layersDir, npminstall.LayerNameNodeModules),
						SharedEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            false,
						Launch:           false,
						Cache:            false,
						Metadata:         nil,
					},
				},
			}))

			Expect(projectPathParser.GetCall.Receives.Path).To(Equal(workingDir))
			Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(workingDir))
			Expect(buildProcess.RunCall.CallCount).To(Equal(0))

			link, err := os.Readlink(filepath.Join(workingDir, "node_modules"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layersDir, npminstall.LayerNameNodeModules, "node_modules")))
		})

		context("when BP_NODE_PROJECT_PATH is set", func() {
			it.Before(func() {
				buildProcess.ShouldRunCall.Returns.Run = true
				projectPathParser.GetCall.Returns.ProjectPath = "some-dir"
				Expect(os.MkdirAll(filepath.Join(workingDir, "some-dir", "node_modules"), os.ModePerm))
			})

			it("resolves and calls the build process", func() {
				result, err := build(packit.BuildContext{
					BuildpackInfo: packit.BuildpackInfo{
						SBOMFormats: []string{"application/vnd.cyclonedx+json", "application/spdx+json", "application/vnd.syft+json"},
					},
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Layers[0].Name).To(Equal(npminstall.LayerNameNodeModules))
				Expect(result.Layers[0].Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameNodeModules)))
				Expect(result.Layers[0].SharedEnv).To(Equal(packit.Environment{}))
				Expect(result.Layers[0].BuildEnv).To(Equal(packit.Environment{}))
				Expect(result.Layers[0].LaunchEnv).To(Equal(packit.Environment{}))
				Expect(result.Layers[0].ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
				Expect(result.Layers[0].Build).To(BeFalse())
				Expect(result.Layers[0].Launch).To(BeFalse())
				Expect(result.Layers[0].Cache).To(BeFalse())
				Expect(result.Layers[0].Metadata).To(Equal(
					map[string]interface{}{
						"built_at":  timestamp,
						"cache_sha": "some-sha",
					}))
				Expect(result.Layers[1].Name).To(Equal(npminstall.LayerNameCache))
				Expect(result.Layers[1].Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
				Expect(result.Layers[1].SharedEnv).To(Equal(packit.Environment{}))
				Expect(result.Layers[1].BuildEnv).To(Equal(packit.Environment{}))
				Expect(result.Layers[1].LaunchEnv).To(Equal(packit.Environment{}))
				Expect(result.Layers[1].ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
				Expect(result.Layers[1].Build).To(BeFalse())
				Expect(result.Layers[1].Launch).To(BeFalse())
				Expect(result.Layers[1].Cache).To(BeFalse())

				Expect(result.Layers[0].SBOM.Formats()).To(Equal([]packit.SBOMFormat{
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

				Expect(projectPathParser.GetCall.Receives.Path).To(Equal(workingDir))
				Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(filepath.Join(workingDir, "some-dir")))
				Expect(environment.ConfigureCall.CallCount).To(Equal(1))
				Expect(environment.ConfigureCall.Receives.Layer.Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameNodeModules)))

				Expect(processLayerDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameNodeModules)))
				Expect(processCacheDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
				Expect(processWorkingDir).To(Equal(filepath.Join(workingDir, "some-dir")))
			})
		})

		context("failure cases", func() {
			context("when the node_modules directory cannot be removed", func() {
				it.Before(func() {
					Expect(os.Chmod(workingDir, 0000))
				})

				it.After(func() {
					Expect(os.Chmod(workingDir, os.ModePerm))
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
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

	context("when the cache layer directory is empty", func() {
		it.Before(func() {
			buildProcess.RunCall.Stub = func(ld, cd, wd string) error {
				err := os.MkdirAll(cd, os.ModePerm)
				if err != nil {
					return err
				}

				return nil
			}

			build = npminstall.Build(projectPathParser, buildManager, clock, environment, scribe.NewLogger(buffer), sbomGenerator)
		})

		it("filters out empty layers", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Layers[0].Name).To(Equal(npminstall.LayerNameNodeModules))
			Expect(result.Layers[0].Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameNodeModules)))
			Expect(result.Layers[0].SharedEnv).To(Equal(packit.Environment{}))
			Expect(result.Layers[0].Build).To(BeFalse())
			Expect(result.Layers[0].Cache).To(BeFalse())
			Expect(result.Layers[0].Metadata).To(Equal(
				map[string]interface{}{
					"built_at":  timestamp,
					"cache_sha": "some-sha",
				}))
		})
	})

	context("when the cache layer directory does not exist", func() {
		it.Before(func() {
			buildProcess.RunCall.Stub = func(ld, cd, wd string) error { return nil }

			build = npminstall.Build(projectPathParser, buildManager, clock, environment, scribe.NewLogger(buffer), sbomGenerator)
		})

		it("filters out empty layers", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Layers[0].Name).To(Equal(npminstall.LayerNameNodeModules))
			Expect(result.Layers[0].Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameNodeModules)))
			Expect(result.Layers[0].SharedEnv).To(Equal(packit.Environment{}))
			Expect(result.Layers[0].BuildEnv).To(Equal(packit.Environment{}))
			Expect(result.Layers[0].LaunchEnv).To(Equal(packit.Environment{}))
			Expect(result.Layers[0].ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(result.Layers[0].Build).To(BeFalse())
			Expect(result.Layers[0].Launch).To(BeFalse())
			Expect(result.Layers[0].Cache).To(BeFalse())
			Expect(result.Layers[0].Metadata).To(Equal(
				map[string]interface{}{
					"built_at":  timestamp,
					"cache_sha": "some-sha",
				}))
		})
	})

	context("failure cases", func() {
		context("when the node_modules layer cannot be fetched", func() {
			it.Before(func() {
				_, err := os.Create(filepath.Join(layersDir, "modules.toml"))
				Expect(err).NotTo(HaveOccurred())
				Expect(os.Chmod(filepath.Join(layersDir, "modules.toml"), 0000)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

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
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
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
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).To(MatchError("failed to resolve build process"))
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
				Expect(os.Mkdir(filepath.Join(layersDir, npminstall.LayerNameNodeModules), os.ModePerm)).To(Succeed())

				_, err := os.OpenFile(filepath.Join(layersDir, npminstall.LayerNameNodeModules, "some-file"), os.O_CREATE, 0000)
				Expect(err).NotTo(HaveOccurred())

				Expect(os.Chmod(filepath.Join(layersDir, npminstall.LayerNameNodeModules), 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(filepath.Join(layersDir, npminstall.LayerNameNodeModules), os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
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
				buildProcess.RunCall.Stub = func(string, string, string) error {
					return errors.New("given build process failed")
				}
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).To(MatchError("given build process failed"))
			})
		})

		context("when the project path parser provided fails", func() {
			it.Before(func() {
				projectPathParser.GetCall.Returns.Err = errors.New("some-error")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).To(MatchError("some-error"))
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
					BuildpackInfo: packit.BuildpackInfo{
						SBOMFormats: []string{"random-format"},
					},
				})
				Expect(err).To(MatchError("\"random-format\" is not a supported SBOM format"))
			})
		})
	})
}
