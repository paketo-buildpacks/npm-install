package build_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/libjavabuildpack/test"
	"github.com/cloudfoundry/npm-cnb/build"
	"github.com/cloudfoundry/npm-cnb/detect"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
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

		appRoot     string
		cacheLayer  string
		launchLayer string

		appRootNodeModules     string
		cacheLayerNodeModules  string
		launchLayerNodeModules string
	)

	it.Before(func() {
		f = test.NewBuildFactory(t)
		mockCtrl = gomock.NewController(t)
		mockNpm = NewMockModuleInstaller(mockCtrl)

		appRoot = f.Build.Application.Root
		launchLayer = f.Build.Launch.Layer(detect.NPMDependency).Root
		cacheLayer = f.Build.Cache.Layer(detect.NPMDependency).Root

		appRootNodeModules = filepath.Join(appRoot, "node_modules")
		cacheLayerNodeModules = filepath.Join(cacheLayer, "node_modules")
		launchLayerNodeModules = filepath.Join(launchLayer, "node_modules")
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
			Expect(build.CreateLaunchMetadata()).To(Equal(libbuildpack.LaunchMetadata{
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
			err = os.MkdirAll(appRoot, 0777)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(appRoot, "package-lock.json"), []byte("package lock"), 0666)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(appRoot, "package.json"), []byte("package json"), 0666)
			Expect(err).To(BeNil())
		})

		when("when build and launch are not set", func() {
			it("does not install node modules", func() {
				f.AddBuildPlan(t, detect.NPMDependency, libbuildpack.BuildPlanDependency{
					Metadata: libbuildpack.BuildPlanDependencyMetadata{},
				})

				modules, _, err := build.NewModules(f.Build, mockNpm)
				Expect(err).NotTo(HaveOccurred())

				mockNpm.EXPECT().InstallToLayer(gomock.Any(), gomock.Any()).Times(0)
				mockNpm.EXPECT().RebuildLayer(gomock.Any(), gomock.Any()).Times(0)

				err = modules.Contribute()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		when("build is set to true", func() {
			it.Before(func() {
				f.AddBuildPlan(t, detect.NPMDependency, libbuildpack.BuildPlanDependency{
					Metadata: libbuildpack.BuildPlanDependencyMetadata{
						"build": true,
					},
				})
				modules, _, err = build.NewModules(f.Build, mockNpm)
				Expect(err).NotTo(HaveOccurred())

			})

			it("exposes node_modules from the cache via a layer env file", func() {
				mockNpm.EXPECT().InstallToLayer(gomock.Any(), gomock.Any()).Times(1)

				err = modules.Contribute()
				Expect(err).ToNot(HaveOccurred())

				envPath := filepath.Join(cacheLayer, "env", "NODE_PATH")
				Expect(envPath).To(BeAnExistingFile())

				buf, err := ioutil.ReadFile(envPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(buf)).To(Equal(cacheLayerNodeModules))
			})

			when("unvendored", func() {
				it.Before(func() {
					err = ioutil.WriteFile(filepath.Join(appRoot, "package.json"), []byte("package_json"), 0666)
					Expect(err).NotTo(HaveOccurred())
				})
				it("installs node_modules into the cache layer", func() {
					mockNpm.EXPECT().InstallToLayer(appRoot, cacheLayer).Times(1)
					mockNpm.EXPECT().RebuildLayer(gomock.Any(), gomock.Any()).Times(0)
					err = modules.Contribute()
					Expect(err).ToNot(HaveOccurred())
				})
			})

			when("vendored", func() {
				it.Before(func() {
					err = os.MkdirAll(appRootNodeModules, 0777)
					Expect(err).To(BeNil())
				})
				it("it rebuild the modules in the cache layer", func() {
					mockNpm.EXPECT().RebuildLayer(appRoot, cacheLayer).Times(1)
					mockNpm.EXPECT().InstallToLayer(gomock.Any(), gomock.Any()).Times(0)
					err = modules.Contribute()
					Expect(err).ToNot(HaveOccurred())
				})
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

			it("should write the NODE_PATH env var to the launch layer", func() {
				mockNpm.EXPECT().InstallToLayer(gomock.Any(), gomock.Any()).Times(1)
				mockNpm.EXPECT().CleanAndCopyToDst(
					cacheLayerNodeModules,
					launchLayerNodeModules,
				).Times(1)

				err = modules.Contribute()
				Expect(err).ToNot(HaveOccurred())

				envPath := filepath.Join(launchLayer, "profile.d", "NODE_PATH")
				Expect(envPath).To(BeAnExistingFile())

				buf, err := ioutil.ReadFile(envPath)
				Expect(err).ToNot(HaveOccurred())
				Expect(string(buf)).To(Equal(fmt.Sprintf("export NODE_PATH=%s", launchLayerNodeModules)))
			})

			when("when node_modules are NOT vendored", func() {
				when("and there is no launch layer metadata", func() {
					it.Before(func() {
						err = ioutil.WriteFile(filepath.Join(appRoot, "package.json"), []byte("packageJSONcontents"), 0666)
						Expect(err).NotTo(HaveOccurred())
					})

					it("installs node modules and writes metadata", func() {
						mockNpm.EXPECT().InstallToLayer(appRoot, cacheLayer).Times(1)
						mockNpm.EXPECT().CleanAndCopyToDst(cacheLayerNodeModules, launchLayerNodeModules).Times(1)
						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						var metadata build.Metadata
						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})

				when("and there is launch layer metadata that is the same", func() {
					it("does not install node modules or re-write metadata", func() {
						metadata := build.Metadata{SHA256: "152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"}
						f.Build.Launch.Layer(detect.NPMDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().InstallToLayer(appRoot, cacheLayer).Times(0)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})

				when("and there is launch layer metadata that is different", func() {
					it.Before(func() {
						err = ioutil.WriteFile(filepath.Join(appRoot, "package.json"), []byte("newPackageJson"), 0666)
						Expect(err).NotTo(HaveOccurred())
					})

					it("installs node modules and writes metadata", func() {
						metadata := build.Metadata{SHA256: "123456"}
						f.Build.Launch.Layer(detect.NPMDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().InstallToLayer(appRoot, cacheLayer).Times(1)
						mockNpm.EXPECT().CleanAndCopyToDst(cacheLayerNodeModules, launchLayerNodeModules).Times(1)
						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"), "Sha is differant")
					})
				})
			})

			when("when node modules are vendored", func() {
				it.Before(func() {
					err = os.MkdirAll(appRootNodeModules, 0777)
					Expect(err).NotTo(HaveOccurred())

					err = ioutil.WriteFile(filepath.Join(appRoot, "node_modules", "some_module"), []byte("module"), 0666)
					Expect(err).To(BeNil())
				})

				when("and there is no launch layer metadata", func() {
					it("rebuilds the node modules and writes metadata", func() {
						mockNpm.EXPECT().RebuildLayer(appRoot, cacheLayer).Times(1)
						mockNpm.EXPECT().InstallToLayer(appRoot, cacheLayer).Times(0)
						mockNpm.EXPECT().CleanAndCopyToDst(cacheLayerNodeModules, launchLayerNodeModules).Times(1)
						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						var metadata build.Metadata
						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})

				when("and there is launch layer metadata that is the same", func() {
					it("does not rebuild the node modules or re-write metadata", func() {
						metadata := build.Metadata{SHA256: "152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"}
						f.Build.Launch.Layer(detect.NPMDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().RebuildLayer(gomock.Any(), gomock.Any()).Times(0)
						mockNpm.EXPECT().InstallToLayer(appRoot, cacheLayer).Times(0)
						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})

				when("and there is launch layer metadata that is different", func() {
					it("copies node_modules to the cache and launch layers, rebuilds them, and writes metadata", func() {
						metadata := build.Metadata{SHA256: "123456"}
						f.Build.Launch.Layer(detect.NPMDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().RebuildLayer(appRoot, cacheLayer).Times(1)
						mockNpm.EXPECT().CleanAndCopyToDst(cacheLayerNodeModules, launchLayerNodeModules).Times(1)
						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})
			})
		})
	})
}
