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

		layersDir      string
		workingDir     string
		packageManager *fakes.PackageManager
		build          packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		packageManager = &fakes.PackageManager{}

		build = npm.Build(packageManager)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a result that installs node modules", func() {
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
					Name:      "node_modules",
					Path:      filepath.Join(layersDir, "node_modules"),
					SharedEnv: packit.Environment{},
					BuildEnv:  packit.Environment{},
					LaunchEnv: packit.Environment{},
					Build:     false,
					Launch:    true,
					Cache:     false,
				},
			},
			Processes: []packit.Process{
				{
					Type:    "web",
					Command: "npm start",
				},
			},
		}))

		Expect(packageManager.InstallCall.Receives.Dir).To(Equal(workingDir))

		path, err := os.Readlink(filepath.Join(workingDir, "node_modules"))
		Expect(err).NotTo(HaveOccurred())
		Expect(path).To(Equal(filepath.Join(layersDir, "node_modules", "node_modules")))
	})

	context("failure cases", func() {
		context("when the node_modules layer cannot be fetched", func() {
			it.Before(func() {
				_, err := os.Create(filepath.Join(layersDir, "node_modules.toml"))
				Expect(err).NotTo(HaveOccurred())
				Expect(os.Chmod(filepath.Join(layersDir, "node_modules.toml"), 0000)).To(Succeed())
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
				Expect(os.Mkdir(filepath.Join(layersDir, "node_modules"), os.ModePerm)).To(Succeed())

				_, err := os.OpenFile(filepath.Join(layersDir, "node_modules", "some-file"), os.O_CREATE, 0000)
				Expect(err).NotTo(HaveOccurred())

				Expect(os.Chmod(filepath.Join(layersDir, "node_modules"), 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(filepath.Join(layersDir, "node_modules"), os.ModePerm)).To(Succeed())
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

		context("when the node_modules symlink cannot be created", func() {
			it.Before(func() {
				Expect(os.Chmod(workingDir, 0000)).To(Succeed())
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

		context("when the package manager cannot install", func() {
			it.Before(func() {
				packageManager.InstallCall.Returns.Error = errors.New("failed to install")
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
				Expect(err).To(MatchError("failed to install"))
			})
		})
	})
}
