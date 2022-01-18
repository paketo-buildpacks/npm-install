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
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/packit/v2/servicebindings"

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
		processNpmrcPath  string

		timestamp string

		projectPathParser *fakes.PathParser
		buildProcess      *fakes.BuildProcess
		buildManager      *fakes.BuildManager
		bindingResolver   *fakes.BindingResolver
		environment       *fakes.EnvironmentConfig
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

		bindingResolver = &fakes.BindingResolver{}

		buildProcess = &fakes.BuildProcess{}
		buildProcess.ShouldRunCall.Returns.Run = true
		buildProcess.ShouldRunCall.Returns.Sha = "some-sha"
		buildProcess.RunCall.Stub = func(ld, cd, wd, rc string) error {
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
			processNpmrcPath = rc

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

		build = npminstall.Build(
			projectPathParser,
			bindingResolver,
			buildManager,
			clock,
			environment,
			logger,
		)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("resolves and calls the build process", func() {
		result, err := build(packit.BuildContext{
			Platform: packit.Platform{
				Path: "some-platform-path",
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
					Metadata: map[string]interface{}{
						"built_at":  timestamp,
						"cache_sha": "some-sha",
					},
				}, {
					Name:             npminstall.LayerNameCache,
					Path:             filepath.Join(layersDir, npminstall.LayerNameCache),
					SharedEnv:        packit.Environment{},
					BuildEnv:         packit.Environment{},
					LaunchEnv:        packit.Environment{},
					ProcessLaunchEnv: map[string]packit.Environment{},
					Build:            false,
					Launch:           false,
					Cache:            false,
				},
			},
		}))

		Expect(projectPathParser.GetCall.Receives.Path).To(Equal(workingDir))
		Expect(bindingResolver.ResolveCall.Receives.Typ).To(Equal("npmrc"))
		Expect(bindingResolver.ResolveCall.Receives.Provider).To(Equal(""))
		Expect(bindingResolver.ResolveCall.Receives.PlatformDir).To(Equal("some-platform-path"))
		Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(workingDir))
		Expect(environment.ConfigureCall.CallCount).To(Equal(1))
		Expect(environment.ConfigureCall.Receives.Layer.Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameNodeModules)))

		Expect(processLayerDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameNodeModules)))
		Expect(processCacheDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
		Expect(processWorkingDir).To(Equal(workingDir))
		Expect(processNpmrcPath).To(Equal(""))
	})

	context("when node_modules is required at build and launch", func() {
		it("resolves and calls the build process", func() {
			result, err := build(packit.BuildContext{
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
			Expect(result).To(Equal(packit.BuildResult{
				Layers: []packit.Layer{
					{
						Name:             npminstall.LayerNameNodeModules,
						Path:             filepath.Join(layersDir, npminstall.LayerNameNodeModules),
						SharedEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            true,
						Launch:           true,
						Cache:            true,
						Metadata: map[string]interface{}{
							"built_at":  timestamp,
							"cache_sha": "some-sha",
						},
					}, {
						Name:             npminstall.LayerNameCache,
						Path:             filepath.Join(layersDir, npminstall.LayerNameCache),
						SharedEnv:        packit.Environment{},
						BuildEnv:         packit.Environment{},
						LaunchEnv:        packit.Environment{},
						ProcessLaunchEnv: map[string]packit.Environment{},
						Build:            true,
						Launch:           true,
						Cache:            true,
					},
				},
			}))

			Expect(projectPathParser.GetCall.Receives.Path).To(Equal(workingDir))
			Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(workingDir))

			Expect(processLayerDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameNodeModules)))
			Expect(processCacheDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
			Expect(processWorkingDir).To(Equal(workingDir))
		})
	})

	context("when npmrc service bindings are detected", func() {
		context("when one npmrc binding is detected", func() {
			it.Before(func() {
				buffer = bytes.NewBuffer(nil)
				logger := scribe.NewLogger(buffer)

				bindingResolver.ResolveCall.Returns.BindingSlice = []servicebindings.Binding{
					servicebindings.Binding{
						Name: "some-binding",
						Type: "npmrc",
						Path: "some-binding-path",
						Entries: map[string]*servicebindings.Entry{
							".npmrc": servicebindings.NewEntry("some-entry-path"),
						},
					},
				}
				build = npminstall.Build(
					projectPathParser,
					bindingResolver,
					buildManager,
					clock,
					environment,
					logger,
				)
			})

			it("passes the path to the .npmrc to the build process and env configuration", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})

				Expect(err).NotTo(HaveOccurred())
				Expect(buildProcess.ShouldRunCall.Receives.NpmrcPath).To(Equal("some-binding-path/.npmrc"))
				Expect(buildProcess.RunCall.Receives.NpmrcPath).To(Equal("some-binding-path/.npmrc"))
				Expect(environment.ConfigureCall.Receives.NpmrcPath).To(Equal("some-binding-path/.npmrc"))
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
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})

					Expect(err).NotTo(HaveOccurred())
					Expect(buildProcess.ShouldRunCall.Receives.NpmrcPath).To(Equal("some/path/.npmrc"))
					Expect(buildProcess.RunCall.Receives.NpmrcPath).To(Equal("some/path/.npmrc"))
					Expect(environment.ConfigureCall.Receives.NpmrcPath).To(Equal("some/path/.npmrc"))
				})
			})

			context("when the binding does not contain an .npmrc entry", func() {
				it.Before(func() {
					buffer = bytes.NewBuffer(nil)
					logger := scribe.NewLogger(buffer)

					bindingResolver.ResolveCall.Returns.BindingSlice = []servicebindings.Binding{
						servicebindings.Binding{
							Name: "some-binding",
							Type: "npmrc",
							Path: "some-binding-path",
							Entries: map[string]*servicebindings.Entry{
								"some-unrelated-entry": servicebindings.NewEntry("some-entry-path"),
							},
						},
					}
					build = npminstall.Build(
						projectPathParser,
						bindingResolver,
						buildManager,
						clock,
						environment,
						logger,
					)
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
					Expect(err).To(MatchError(ContainSubstring("binding of type 'npmrc' does not contain required entry '.npmrc'")))
				})
			})
		})

		context("when more than one npmrc binding is found", func() {
			it.Before(func() {
				buffer = bytes.NewBuffer(nil)
				logger := scribe.NewLogger(buffer)

				bindingResolver.ResolveCall.Returns.BindingSlice = []servicebindings.Binding{
					servicebindings.Binding{
						Name: "some-binding",
						Type: "npmrc",
					},
					servicebindings.Binding{
						Name: "some-other-binding",
						Type: "npmrc",
					},
				}
				build = npminstall.Build(
					projectPathParser,
					bindingResolver,
					buildManager,
					clock,
					environment,
					logger,
				)
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
				Expect(err).To(MatchError(ContainSubstring("binding resolver found more than one binding of type 'npmrc'")))
			})
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
							Metadata: map[string]interface{}{
								"built_at":  timestamp,
								"cache_sha": "some-sha",
							},
						}, {
							Name:             npminstall.LayerNameCache,
							Path:             filepath.Join(layersDir, npminstall.LayerNameCache),
							SharedEnv:        packit.Environment{},
							BuildEnv:         packit.Environment{},
							LaunchEnv:        packit.Environment{},
							ProcessLaunchEnv: map[string]packit.Environment{},
							Build:            false,
							Launch:           false,
							Cache:            false,
						},
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
			buildProcess.RunCall.Stub = func(ld, cd, wd, rc string) error {
				err := os.MkdirAll(cd, os.ModePerm)
				if err != nil {
					return err
				}

				return nil
			}

			build = npminstall.Build(
				projectPathParser,
				bindingResolver,
				buildManager,
				clock,
				environment,
				scribe.NewLogger(buffer),
			)
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
			Expect(result.Layers).To(Equal([]packit.Layer{
				{
					Name: npminstall.LayerNameNodeModules,
					Path: filepath.Join(layersDir, npminstall.LayerNameNodeModules),

					SharedEnv:        packit.Environment{},
					BuildEnv:         packit.Environment{},
					LaunchEnv:        packit.Environment{},
					ProcessLaunchEnv: map[string]packit.Environment{},
					Build:            false,
					Launch:           false,
					Cache:            false,
					Metadata: map[string]interface{}{
						"built_at":  timestamp,
						"cache_sha": "some-sha",
					},
				},
			}))
		})
	})

	context("when the cache layer directory does not exist", func() {
		it.Before(func() {
			buildProcess.RunCall.Stub = func(ld, cd, wd, rc string) error { return nil }

			build = npminstall.Build(
				projectPathParser,
				bindingResolver,
				buildManager,
				clock,
				environment,
				scribe.NewLogger(buffer),
			)
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
			Expect(result.Layers).To(Equal([]packit.Layer{
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
					Metadata: map[string]interface{}{
						"built_at":  timestamp,
						"cache_sha": "some-sha",
					},
				},
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
				buildProcess.RunCall.Stub = func(string, string, string, string) error {
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
		context("when service binding resolution fails", func() {
			it.Before(func() {
				bindingResolver.ResolveCall.Returns.Error = errors.New("some-bindings-error")
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
				Expect(err).To(MatchError("some-bindings-error"))
			})
		})
	})
}
