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
		mockCtrl = gomock.NewController(GinkgoT())
		f = test.NewBuildFactory(T)
		mockNpm = NewMockModuleInstaller(mockCtrl)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("CreateLaunchMetadata", func() {
		It("returns launch metadata for running with npm", func() {
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

	Context("NewModules", func() {
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

	Context("Contribute", func() {
		It("does not install node modules to the cache or launch layer when build and launch are not set", func() {
			f.AddBuildPlan(T, detect.ModulesDependency, libbuildpack.BuildPlanDependency{
				Metadata: libbuildpack.BuildPlanDependencyMetadata{},
			})

			modules, _, err := build.NewModules(f.Build, mockNpm)
			Expect(err).NotTo(HaveOccurred())

			mockNpm.EXPECT().Install(gomock.Any()).Times(0)
			err = modules.Contribute()
			Expect(err).NotTo(HaveOccurred())
		})

		Context("When node_modules are NOT vendored", func() {
			It("installs node modules to the cache layer when build is true", func() {
				f.AddBuildPlan(T, detect.ModulesDependency, libbuildpack.BuildPlanDependency{
					Metadata: libbuildpack.BuildPlanDependencyMetadata{
						"build": true,
					},
				})

				modules, _, err := build.NewModules(f.Build, mockNpm)
				Expect(err).NotTo(HaveOccurred())

				mockNpm.EXPECT().Install(f.Build.Application.Root).Do(func(dir string) {
					err = os.MkdirAll(filepath.Join(dir, "node_modules"), 0777)
					Expect(err).To(BeNil())
					err = ioutil.WriteFile(filepath.Join(dir, "node_modules", "some_module"), []byte("module"), 0666)
					Expect(err).To(BeNil())
				})

				err = modules.Contribute()
				Expect(err).NotTo(HaveOccurred())

				depCacheLayer := filepath.Join(f.Build.Cache.Root, "modules")
				Expect(filepath.Join(depCacheLayer, "node_modules", "some_module")).To(BeAnExistingFile())
			})

			It("installs node modules to the launch layer when launch is true", func() {
				f.AddBuildPlan(T, detect.ModulesDependency, libbuildpack.BuildPlanDependency{
					Metadata: libbuildpack.BuildPlanDependencyMetadata{
						"launch": true,
					},
				})

				modules, _, err := build.NewModules(f.Build, mockNpm)
				Expect(err).NotTo(HaveOccurred())

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
			})
		})

		Context("when node modules are vendored", func() {
			BeforeEach(func() {
				err = os.MkdirAll(filepath.Join(f.Build.Application.Root, "node_modules"), 0777)
				Expect(err).NotTo(HaveOccurred())
				err = ioutil.WriteFile(filepath.Join(f.Build.Application.Root, "node_modules", "some_module"), []byte("module"), 0666)
				Expect(err).To(BeNil())
			})

			It("rebuilds the node modules and copies them to the cache layer when build is true", func() {
				f.AddBuildPlan(T, detect.ModulesDependency, libbuildpack.BuildPlanDependency{
					Metadata: libbuildpack.BuildPlanDependencyMetadata{
						"build": true,
					},
				})

				modules, _, err := build.NewModules(f.Build, mockNpm)
				Expect(err).NotTo(HaveOccurred())

				mockNpm.EXPECT().Rebuild(f.Build.Application.Root).Return(nil).Times(1)

				err = modules.Contribute()
				Expect(err).NotTo(HaveOccurred())

				depCacheLayer := filepath.Join(f.Build.Cache.Root, "modules")
				Expect(filepath.Join(depCacheLayer, "node_modules", "some_module")).To(BeAnExistingFile())
			})

			It("rebuilds the node modules and copies them to the launch layer when launch is true", func() {
				f.AddBuildPlan(T, detect.ModulesDependency, libbuildpack.BuildPlanDependency{
					Metadata: libbuildpack.BuildPlanDependencyMetadata{
						"launch": true,
					},
				})

				modules, _, err := build.NewModules(f.Build, mockNpm)
				Expect(err).NotTo(HaveOccurred())

				mockNpm.EXPECT().Rebuild(f.Build.Application.Root).Return(nil).Times(1)

				err = modules.Contribute()
				Expect(err).NotTo(HaveOccurred())

				depLaunchLayer := filepath.Join(f.Build.Launch.Root, "modules")
				Expect(filepath.Join(depLaunchLayer, "node_modules", "some_module")).To(BeAnExistingFile())
			})
		})
	})
})
