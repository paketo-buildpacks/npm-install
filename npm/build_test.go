package npm_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/npm-cnb/npm"
	"github.com/cloudfoundry/npm-cnb/npm/fakes"
	"github.com/cloudfoundry/packit"
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

		buildManager *fakes.BuildManager
		build        packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		buildManager = &fakes.BuildManager{}
		buildManager.ResolveCall.Returns.BuildProcess = func(ld, cd, wd string) error {
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

		build = npm.Build(buildManager)
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
					Name:      npm.LayerNameNodeModules,
					Path:      filepath.Join(layersDir, npm.LayerNameNodeModules),
					SharedEnv: packit.Environment{},
					BuildEnv:  packit.Environment{},
					LaunchEnv: packit.Environment{},
					Build:     false,
					Launch:    true,
					Cache:     false,
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

	context("when the cache layer directory is empty", func() {
		it.Before(func() {
			buildManager.ResolveCall.Returns.BuildProcess = func(ld, cd, wd string) error {
				err := os.MkdirAll(cd, os.ModePerm)
				if err != nil {
					return err
				}

				return nil
			}

			build = npm.Build(buildManager)
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
					Name:      npm.LayerNameNodeModules,
					Path:      filepath.Join(layersDir, npm.LayerNameNodeModules),
					SharedEnv: packit.Environment{},
					BuildEnv:  packit.Environment{},
					LaunchEnv: packit.Environment{},
					Build:     false,
					Launch:    true,
					Cache:     false,
				},
			}))
		})
	})

	context("when the cache layer directory does not exist", func() {
		it.Before(func() {
			buildManager.ResolveCall.Returns.BuildProcess = func(ld, cd, wd string) error { return nil }

			build = npm.Build(buildManager)
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
					Name:      npm.LayerNameNodeModules,
					Path:      filepath.Join(layersDir, npm.LayerNameNodeModules),
					SharedEnv: packit.Environment{},
					BuildEnv:  packit.Environment{},
					LaunchEnv: packit.Environment{},
					Build:     false,
					Launch:    true,
					Cache:     false,
				},
			}))
		})
	})

	context("failure cases", func() {
		context("when the node_modules layer cannot be fetched", func() {
			it.Before(func() {
				_, err := os.Create(filepath.Join(layersDir, "modules_layer.toml"))
				Expect(err).NotTo(HaveOccurred())
				Expect(os.Chmod(filepath.Join(layersDir, "modules_layer.toml"), 0000)).To(Succeed())
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

		context("when the build process provided fails", func() {
			it.Before(func() {
				buildManager.ResolveCall.Returns.BuildProcess = func(string, string, string) error {
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
