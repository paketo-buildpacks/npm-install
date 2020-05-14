package npm_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/paketo-buildpacks/npm/npm"
	"github.com/paketo-buildpacks/npm/npm/fakes"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/scribe"
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

		buildProcess *fakes.BuildProcess
		buildManager *fakes.BuildManager
		clock        npm.Clock
		build        packit.BuildFunc

		buffer *bytes.Buffer
	)

	it.Before(func() {
		var err error
		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

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
		clock = npm.NewClock(func() time.Time {
			return now
		})
		timestamp = now.Format(time.RFC3339Nano)

		buildManager = &fakes.BuildManager{}
		buildManager.ResolveCall.Returns.BuildProcess = buildProcess

		buffer = bytes.NewBuffer(nil)
		logger := scribe.NewLogger(buffer)

		build = npm.Build(buildManager, clock, &logger)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
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
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{Name: "node_modules"},
				},
			},
			Layers: []packit.Layer{
				{
					Name: npm.LayerNameNodeModules,
					Path: filepath.Join(layersDir, npm.LayerNameNodeModules),
					SharedEnv: packit.Environment{
						"PATH.append": filepath.Join(layersDir, npm.LayerNameNodeModules, "node_modules", ".bin"),
						"PATH.delim":  string(os.PathListSeparator),
					},
					BuildEnv: packit.Environment{},
					LaunchEnv: packit.Environment{
						"NPM_CONFIG_LOGLEVEL.override":   "error",
						"NPM_CONFIG_PRODUCTION.override": "true",
					},
					Build:  false,
					Launch: true,
					Cache:  false,
					Metadata: map[string]interface{}{
						"built_at":  timestamp,
						"cache_sha": "some-sha",
					},
				}, {
					Name:      npm.LayerNameCache,
					Path:      filepath.Join(layersDir, npm.LayerNameCache),
					SharedEnv: packit.Environment{},
					BuildEnv:  packit.Environment{},
					LaunchEnv: packit.Environment{},
					Build:     false,
					Launch:    false,
					Cache:     true,
				},
			},
			Processes: []packit.Process{
				{
					Type:    "web",
					Command: "npm start",
				},
			},
		}))

		Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(workingDir))

		Expect(processLayerDir).To(Equal(filepath.Join(layersDir, npm.LayerNameNodeModules)))
		Expect(processCacheDir).To(Equal(filepath.Join(layersDir, npm.LayerNameCache)))
		Expect(processWorkingDir).To(Equal(workingDir))
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
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
				Layers: []packit.Layer{
					{
						Name:      npm.LayerNameNodeModules,
						Path:      filepath.Join(layersDir, npm.LayerNameNodeModules),
						SharedEnv: packit.Environment{},
						BuildEnv:  packit.Environment{},
						LaunchEnv: packit.Environment{},
						Build:     false,
						Launch:    true,
						Cache:     false,
						Metadata:  nil,
					},
				},
				Processes: []packit.Process{
					{
						Type:    "web",
						Command: "npm start",
					},
				},
			}))

			Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(workingDir))
			Expect(buildProcess.RunCall.CallCount).To(Equal(0))

			link, err := os.Readlink(filepath.Join(workingDir, "node_modules"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layersDir, npm.LayerNameNodeModules, "node_modules")))
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

			logger := scribe.NewLogger(buffer)
			build = npm.Build(buildManager, clock, &logger)
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
					Name: npm.LayerNameNodeModules,
					Path: filepath.Join(layersDir, npm.LayerNameNodeModules),

					SharedEnv: packit.Environment{
						"PATH.append": filepath.Join(layersDir, npm.LayerNameNodeModules, "node_modules", ".bin"),
						"PATH.delim":  string(os.PathListSeparator),
					},
					BuildEnv: packit.Environment{},
					LaunchEnv: packit.Environment{
						"NPM_CONFIG_LOGLEVEL.override":   "error",
						"NPM_CONFIG_PRODUCTION.override": "true",
					},
					Build:  false,
					Launch: true,
					Cache:  false,
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
			buildProcess.RunCall.Stub = func(ld, cd, wd string) error { return nil }

			logger := scribe.NewLogger(buffer)
			build = npm.Build(buildManager, clock, &logger)
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
					Name: npm.LayerNameNodeModules,
					Path: filepath.Join(layersDir, npm.LayerNameNodeModules),
					SharedEnv: packit.Environment{
						"PATH.append": filepath.Join(layersDir, npm.LayerNameNodeModules, "node_modules", ".bin"),
						"PATH.delim":  string(os.PathListSeparator),
					},
					BuildEnv: packit.Environment{},
					LaunchEnv: packit.Environment{
						"NPM_CONFIG_LOGLEVEL.override":   "error",
						"NPM_CONFIG_PRODUCTION.override": "true",
					},
					Build:  false,
					Launch: true,
					Cache:  false,
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
				Expect(os.Mkdir(filepath.Join(layersDir, npm.LayerNameNodeModules), os.ModePerm)).To(Succeed())

				_, err := os.OpenFile(filepath.Join(layersDir, npm.LayerNameNodeModules, "some-file"), os.O_CREATE, 0000)
				Expect(err).NotTo(HaveOccurred())

				Expect(os.Chmod(filepath.Join(layersDir, npm.LayerNameNodeModules), 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(filepath.Join(layersDir, npm.LayerNameNodeModules), os.ModePerm)).To(Succeed())
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
	})
}
