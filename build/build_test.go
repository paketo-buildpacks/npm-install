package build_test

import (
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/npm-cnb/detect"

	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libjavabuildpack/test"
	"github.com/cloudfoundry/npm-cnb/build"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=build.go -destination=mocks_test.go -package=build_test

func TestUnitBuild(t *testing.T) {
	RegisterTestingT(t)
	spec.Run(t, "Build", testBuild, spec.Report(report.Terminal{}))
}

func testBuild(t *testing.T, when spec.G, it spec.S) {
	var (
		mockCtrl *gomock.Controller
		mockNpm  *MockModuleInstaller
		modules  build.Modules
		f        test.BuildFactory
		err      error
	)

	it.Before(func() {
		f = test.NewBuildFactory(t)

		mockCtrl = gomock.NewController(t)
		mockNpm = NewMockModuleInstaller(mockCtrl)
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("NewModules", func() {
		it("returns true if a build plan exists", func() {
			f.AddBuildPlan(t, detect.NPMDependency, libbuildpack.BuildPlanDependency{})

			_, ok, err := build.NewModules(f.Build, mockNpm)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		it("returns false if a build plan does not exist", func() {
			_, ok, err := build.NewModules(f.Build, mockNpm)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})

	when("CreateLaunchMetadata", func() {
		it("returns launch metadata for running with npm", func() {
			modules, _, _ := build.NewModules(f.Build, mockNpm)

			Expect(modules.CreateLaunchMetadata()).To(Equal(libbuildpack.LaunchMetadata{
				Processes: libbuildpack.Processes{
					libbuildpack.Process{
						Type:    "web",
						Command: "npm start",
					},
				},
			}))
		})
	})

	when("Contribute", func() {
		it.Before(func() {
			err = os.MkdirAll(f.Build.Application.Root, 0777)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(f.Build.Application.Root, "package-lock.json"), []byte("package lock"), 0666)
			Expect(err).To(BeNil())
		})

		when("when build and launch are not set", func() {
			it("does not install node modules", func() {
				f.AddBuildPlan(t, detect.NPMDependency, libbuildpack.BuildPlanDependency{
					Metadata: libbuildpack.BuildPlanDependencyMetadata{},
				})

				modules, _, err := build.NewModules(f.Build, mockNpm)
				Expect(err).NotTo(HaveOccurred())

				mockNpm.EXPECT().Install(gomock.Any()).Times(0)

				err = modules.Contribute()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		when("when launch is set to true", func() {
			it.Before(func() {
				f.AddBuildPlan(t, detect.NPMDependency, libbuildpack.BuildPlanDependency{
					Metadata: libbuildpack.BuildPlanDependencyMetadata{
						"launch": true,
					},
				})

				modules, _, err = build.NewModules(f.Build, mockNpm)
				Expect(err).NotTo(HaveOccurred())
			})

			when("when node_modules are NOT vendored", func() {
				when("and there is no layer metadata", func() {
					it("installs node modules and writes metadata", func() {
						mockNpm.EXPECT().Install(f.Build.Application.Root).Do(func(dir string) {
							err = os.MkdirAll(filepath.Join(dir, "node_modules"), 0777)
							Expect(err).ToNot(HaveOccurred())

							err = ioutil.WriteFile(filepath.Join(dir, "node_modules", "some_module"), []byte("module"), 0666)
							Expect(err).ToNot(HaveOccurred())
						})

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						depLaunchLayer := filepath.Join(f.Build.Launch.Root, detect.NPMDependency)
						Expect(filepath.Join(depLaunchLayer, "node_modules", "some_module")).To(BeAnExistingFile())

						linkPath, err := os.Readlink(filepath.Join(f.Build.Application.Root, "node_modules"))
						Expect(err).NotTo(HaveOccurred())
						Expect(linkPath).To(Equal(filepath.Join(depLaunchLayer, "node_modules")))

						var metadata build.Metadata
						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})

				when("and there is layer metadata that is the same", func() {
					it("does not install node modules", func() {
						metadata := build.Metadata{SHA256: "152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"}
						f.Build.Launch.Layer(detect.NPMDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().Install(f.Build.Application.Root).Times(0)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())
					})
				})

				when("and there is layer metadata that is different", func() {
					it("installs node modules and writes metadata", func() {
						metadata := build.Metadata{SHA256: "123456"}
						f.Build.Launch.Layer(detect.NPMDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().Install(f.Build.Application.Root).Do(func(dir string) {
							err = os.MkdirAll(filepath.Join(dir, "node_modules"), 0777)
							Expect(err).ToNot(HaveOccurred())

							err = ioutil.WriteFile(filepath.Join(dir, "node_modules", "some_module"), []byte("module"), 0666)
							Expect(err).ToNot(HaveOccurred())
						}).Times(1)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						depLaunchLayer := filepath.Join(f.Build.Launch.Root, detect.NPMDependency)
						Expect(filepath.Join(depLaunchLayer, "node_modules", "some_module")).To(BeAnExistingFile())

						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})
			})

			when("when node modules are vendored", func() {
				it.Before(func() {
					err = os.MkdirAll(filepath.Join(f.Build.Application.Root, "node_modules"), 0777)
					Expect(err).NotTo(HaveOccurred())

					err = ioutil.WriteFile(filepath.Join(f.Build.Application.Root, "node_modules", "some_module"), []byte("module"), 0666)
					Expect(err).To(BeNil())
				})

				when("and there is no layer metadata", func() {
					it("rebuilds the node modules and writes metadata", func() {
						mockNpm.EXPECT().Rebuild(f.Build.Application.Root).Return(nil).Times(1)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						depLaunchLayer := filepath.Join(f.Build.Launch.Root, detect.NPMDependency)
						Expect(filepath.Join(depLaunchLayer, "node_modules", "some_module")).To(BeAnExistingFile())

						var metadata build.Metadata
						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})

				when("and there is layer metadata that is the same", func() {
					it("does not rebuild the node modules", func() {
						metadata := build.Metadata{SHA256: "152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"}
						f.Build.Launch.Layer(detect.NPMDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().Rebuild(f.Build.Application.Root).Times(0)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())
					})
				})

				when("and there is layer metadata that is different", func() {
					it("rebuilds the node_modules and writes metadata", func() {
						metadata := build.Metadata{SHA256: "123456"}
						f.Build.Launch.Layer(detect.NPMDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().Rebuild(f.Build.Application.Root).Times(1)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						depLaunchLayer := filepath.Join(f.Build.Launch.Root, detect.NPMDependency)
						Expect(filepath.Join(depLaunchLayer, "node_modules", "some_module")).To(BeAnExistingFile())

						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})
			})
		})
	})
}
