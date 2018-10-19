package build_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/npm-cnb/detect"

	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libjavabuildpack/test"
	"github.com/cloudfoundry/npm-cnb/build"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=build.go -destination=mocks_test.go -package=build_test

var _ = Describe("Build", func() {
	var (
		mockCtrl *gomock.Controller
		mockNpm  *MockModuleInstaller
		modules  build.Modules
		f        test.BuildFactory
		err      error
	)

	BeforeEach(func() {
		f = test.NewBuildFactory(T)

		mockCtrl = gomock.NewController(GinkgoT())
		mockNpm = NewMockModuleInstaller(mockCtrl)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("NewModules", func() {
		It("returns true if a build plan exists", func() {
			f.AddBuildPlan(T, detect.ModulesDependency, libbuildpack.BuildPlanDependency{})

			_, ok, err := build.NewModules(f.Build, mockNpm)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeTrue())
		})

		It("returns false if a build plan does not exist", func() {
			_, ok, err := build.NewModules(f.Build, mockNpm)
			Expect(err).NotTo(HaveOccurred())
			Expect(ok).To(BeFalse())
		})
	})

	Describe("CreateLaunchMetadata", func() {
		It("returns launch metadata for running with npm", func() {
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

	Describe("Contribute", func() {
		BeforeEach(func() {
			err = os.MkdirAll(f.Build.Application.Root, 0777)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(f.Build.Application.Root, "package-lock.json"), []byte("package lock"), 0666)
			Expect(err).To(BeNil())
		})

		Context("when build and launch are not set", func() {
			It("does not install node modules", func() {
				f.AddBuildPlan(T, detect.ModulesDependency, libbuildpack.BuildPlanDependency{
					Metadata: libbuildpack.BuildPlanDependencyMetadata{},
				})

				modules, _, err := build.NewModules(f.Build, mockNpm)
				Expect(err).NotTo(HaveOccurred())

				mockNpm.EXPECT().Install(gomock.Any()).Times(0)

				err = modules.Contribute()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when launch is set to true", func() {
			BeforeEach(func() {
				f.AddBuildPlan(T, detect.ModulesDependency, libbuildpack.BuildPlanDependency{
					Metadata: libbuildpack.BuildPlanDependencyMetadata{
						"launch": true,
					},
				})

				modules, _, err = build.NewModules(f.Build, mockNpm)
				Expect(err).NotTo(HaveOccurred())
			})

			Context("when node_modules are NOT vendored", func() {
				Context("and there is no layer metadata", func() {
					It("installs node modules and writes metadata", func() {
						mockNpm.EXPECT().Install(f.Build.Application.Root).Do(func(dir string) {
							err = os.MkdirAll(filepath.Join(dir, "node_modules"), 0777)
							Expect(err).ToNot(HaveOccurred())

							err = ioutil.WriteFile(filepath.Join(dir, "node_modules", "some_module"), []byte("module"), 0666)
							Expect(err).ToNot(HaveOccurred())
						})

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						depLaunchLayer := filepath.Join(f.Build.Launch.Root, "modules")
						Expect(filepath.Join(depLaunchLayer, "node_modules", "some_module")).To(BeAnExistingFile())

						linkPath, err := os.Readlink(filepath.Join(f.Build.Application.Root, "node_modules"))
						Expect(err).NotTo(HaveOccurred())
						Expect(linkPath).To(Equal(filepath.Join(depLaunchLayer, "node_modules")))

						var metadata build.Metadata
						f.Build.Launch.Layer(detect.ModulesDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})

				Context("and there is layer metadata that is the same", func() {
					It("does not install node modules", func() {
						metadata := build.Metadata{SHA256: "152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"}
						f.Build.Launch.Layer(detect.ModulesDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().Install(f.Build.Application.Root).Times(0)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())
					})
				})

				Context("and there is layer metadata that is different", func() {
					It("installs node modules and writes metadata", func() {
						metadata := build.Metadata{SHA256: "123456"}
						f.Build.Launch.Layer(detect.ModulesDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().Install(f.Build.Application.Root).Do(func(dir string) {
							err = os.MkdirAll(filepath.Join(dir, "node_modules"), 0777)
							Expect(err).ToNot(HaveOccurred())

							err = ioutil.WriteFile(filepath.Join(dir, "node_modules", "some_module"), []byte("module"), 0666)
							Expect(err).ToNot(HaveOccurred())
						}).Times(1)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						depLaunchLayer := filepath.Join(f.Build.Launch.Root, "modules")
						Expect(filepath.Join(depLaunchLayer, "node_modules", "some_module")).To(BeAnExistingFile())

						f.Build.Launch.Layer(detect.ModulesDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})
			})

			Context("when node modules are vendored", func() {
				BeforeEach(func() {
					err = os.MkdirAll(filepath.Join(f.Build.Application.Root, "node_modules"), 0777)
					Expect(err).NotTo(HaveOccurred())

					err = ioutil.WriteFile(filepath.Join(f.Build.Application.Root, "node_modules", "some_module"), []byte("module"), 0666)
					Expect(err).To(BeNil())
				})

				Context("and there is no layer metadata", func() {
					It("rebuilds the node modules and writes metadata", func() {
						mockNpm.EXPECT().Rebuild(f.Build.Application.Root).Return(nil).Times(1)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						depLaunchLayer := filepath.Join(f.Build.Launch.Root, "modules")
						Expect(filepath.Join(depLaunchLayer, "node_modules", "some_module")).To(BeAnExistingFile())

						var metadata build.Metadata
						f.Build.Launch.Layer(detect.ModulesDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})

				Context("and there is layer metadata that is the same", func() {
					It("does not rebuild the node modules", func() {
						metadata := build.Metadata{SHA256: "152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"}
						f.Build.Launch.Layer(detect.ModulesDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().Rebuild(f.Build.Application.Root).Times(0)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())
					})
				})

				Context("and there is layer metadata that is different", func() {
					It("rebuilds the node_modules and writes metadata", func() {
						metadata := build.Metadata{SHA256: "123456"}
						f.Build.Launch.Layer(detect.ModulesDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().Rebuild(f.Build.Application.Root).Times(1)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						depLaunchLayer := filepath.Join(f.Build.Launch.Root, "modules")
						Expect(filepath.Join(depLaunchLayer, "node_modules", "some_module")).To(BeAnExistingFile())

						f.Build.Launch.Layer(detect.ModulesDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})
			})
		})
	})
})
