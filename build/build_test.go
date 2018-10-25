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
		var cacheLayer string

		it.Before(func() {
			err = os.MkdirAll(f.Build.Application.Root, 0777)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(f.Build.Application.Root, "package-lock.json"), []byte("package lock"), 0666)
			Expect(err).To(BeNil())

			err = ioutil.WriteFile(filepath.Join(f.Build.Application.Root, "package.json"), []byte("package json"), 0666)
			Expect(err).To(BeNil())

			cacheLayer = f.Build.Cache.Layer(detect.NPMDependency).Root
			Expect(err).To(BeNil())
		})

		when("when build and launch are not set", func() {
			it("does not install node modules", func() {
				f.AddBuildPlan(t, detect.NPMDependency, libbuildpack.BuildPlanDependency{
					Metadata: libbuildpack.BuildPlanDependencyMetadata{},
				})

				modules, _, err := build.NewModules(f.Build, mockNpm)
				Expect(err).NotTo(HaveOccurred())

				mockNpm.EXPECT().InstallInCache(gomock.Any(), gomock.Any()).Times(0)

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

			it("should write the NODE_PATH env var to the launch layer", func() {
				mockNpm.EXPECT().InstallInCache(gomock.Any(), gomock.Any()).Do(func(src, dir string) {
					err := os.MkdirAll(filepath.Join(dir, "node_modules"), 0777)
					Expect(err).ToNot(HaveOccurred())
				})

				err = modules.Contribute()
				Expect(err).ToNot(HaveOccurred())

				s := f.Build.Launch.Layer(detect.NPMDependency).Root
				envPath := filepath.Join(s, "profile.d", "NODE_PATH")
				Expect(envPath).To(BeAnExistingFile())

				buf, err := ioutil.ReadFile(envPath)
				Expect(err).ToNot(HaveOccurred())

				Expect(string(buf)).To(Equal(fmt.Sprintf("export NODE_PATH=%s", filepath.Join(f.Build.Launch.Layer(detect.NPMDependency).Root, "node_modules"))))
			})

			when("when node_modules are NOT vendored", func() {
				when("and there is no launch layer metadata", func() {
					it.Before(func() {
						err = ioutil.WriteFile(filepath.Join(f.Build.Application.Root, "package.json"), []byte("packageJSONcontents"), 0666)
						Expect(err).NotTo(HaveOccurred())
					})

					it("installs node modules and writes metadata", func() {
						mockNpm.EXPECT().InstallInCache(f.Build.Application.Root, f.Build.Cache.Layer(detect.NPMDependency).Root).Do(func(src, dir string) {
							err = os.MkdirAll(filepath.Join(dir, "node_modules"), 0777)
							Expect(err).ToNot(HaveOccurred())

							err = ioutil.WriteFile(filepath.Join(dir, "node_modules", "some_module"), []byte("module"), 0666)
							Expect(err).ToNot(HaveOccurred())
						})

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						depLaunchLayer := filepath.Join(f.Build.Launch.Root, detect.NPMDependency)
						Expect(filepath.Join(depLaunchLayer, "node_modules", "some_module")).To(BeAnExistingFile())

						var metadata build.Metadata
						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})

				when("and there is launch layer metadata that is the same", func() {
					it("does not install node modules or re-write metadata", func() {
						metadata := build.Metadata{SHA256: "152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"}
						f.Build.Launch.Layer(detect.NPMDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().InstallInCache(f.Build.Application.Root, f.Build.Cache.Layer(detect.NPMDependency).Root).Times(0)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})

				when("and there is launch layer metadata that is different", func() {
					it.Before(func() {
						err = ioutil.WriteFile(filepath.Join(f.Build.Application.Root, "package.json"), []byte("newPackageJson"), 0666)
						Expect(err).NotTo(HaveOccurred())

						// We have to do this to write the existing files below
						err = os.MkdirAll(filepath.Join(cacheLayer), 0777)
						Expect(err).NotTo(HaveOccurred())

						err = ioutil.WriteFile(filepath.Join(cacheLayer, "package.json"), []byte("oldPackageJson"), 0666)
						Expect(err).NotTo(HaveOccurred())

						err = ioutil.WriteFile(filepath.Join(cacheLayer, "package-lock.json"), []byte("old package lock"), 0666)
						Expect(err).NotTo(HaveOccurred())
					})

					it("installs node modules and writes metadata", func() {
						metadata := build.Metadata{SHA256: "123456"}
						f.Build.Launch.Layer(detect.NPMDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().InstallInCache(f.Build.Application.Root, f.Build.Cache.Layer(detect.NPMDependency).Root).Do(func(src, dir string) {
							err = os.MkdirAll(filepath.Join(dir, "node_modules"), 0777)
							Expect(err).ToNot(HaveOccurred())

							err = ioutil.WriteFile(filepath.Join(dir, "node_modules", "some_module"), []byte("module"), 0666)
							Expect(err).ToNot(HaveOccurred())
						}).Times(1)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						depLaunchLayer := filepath.Join(f.Build.Launch.Root, detect.NPMDependency)
						Expect(filepath.Join(depLaunchLayer, "node_modules", "some_module")).To(BeAnExistingFile())

						Expect(filepath.Join(cacheLayer, "package.json")).To(BeAnExistingFile())
						packageContents, err := ioutil.ReadFile(filepath.Join(cacheLayer, "package.json"))
						Expect(packageContents).To(Equal([]byte("newPackageJson")))
						Expect(err).NotTo(HaveOccurred())

						Expect(filepath.Join(cacheLayer, "package-lock.json")).To(BeAnExistingFile())
						lockContents, err := ioutil.ReadFile(filepath.Join(cacheLayer, "package-lock.json"))
						Expect(lockContents).To(Equal([]byte("package lock")))
						Expect(err).NotTo(HaveOccurred())

						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"), "Sha is differant")
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

				when("and there is no launch layer metadata", func() {
					it("rebuilds the node modules and writes metadata", func() {
						mockNpm.EXPECT().Rebuild(f.Build.Cache.Layer(detect.NPMDependency).Root).Return(nil).Times(1)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						depLaunchLayer := filepath.Join(f.Build.Launch.Root, detect.NPMDependency)
						Expect(filepath.Join(depLaunchLayer, "node_modules", "some_module")).To(BeAnExistingFile())

						var metadata build.Metadata
						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})

				when("and there is launch layer metadata that is the same", func() {
					it("does not rebuild the node modules or re-write metadata", func() {
						metadata := build.Metadata{SHA256: "152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"}
						f.Build.Launch.Layer(detect.NPMDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().Rebuild(f.Build.Cache.Layer(detect.NPMDependency).Root).Times(0)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})

				when("and there is launch layer metadata that is different", func() {
					it.Before(func() {
						err = os.MkdirAll(filepath.Join(f.Build.Cache.Layer(detect.NPMDependency).Root, "node_modules"), 0777)
						Expect(err).NotTo(HaveOccurred())

						err = ioutil.WriteFile(filepath.Join(f.Build.Cache.Layer(detect.NPMDependency).Root, "node_modules", "old_module"), []byte("module"), 0666)
						Expect(err).To(BeNil())

					})

					it("copies node_modules to the cache and launch layers, rebuilds them, and writes metadata", func() {
						metadata := build.Metadata{SHA256: "123456"}
						f.Build.Launch.Layer(detect.NPMDependency).WriteMetadata(metadata)

						mockNpm.EXPECT().Rebuild(f.Build.Cache.Layer(detect.NPMDependency).Root).Times(1)

						err = modules.Contribute()
						Expect(err).NotTo(HaveOccurred())

						cacheLayer := filepath.Join(f.Build.Cache.Root, detect.NPMDependency)
						Expect(filepath.Join(cacheLayer, "node_modules", "some_module")).To(BeAnExistingFile())
						Expect(filepath.Join(cacheLayer, "node_modules", "old_module")).ToNot(BeAnExistingFile())

						launchLayer := filepath.Join(f.Build.Launch.Root, detect.NPMDependency)
						Expect(filepath.Join(launchLayer, "node_modules", "some_module")).To(BeAnExistingFile())

						f.Build.Launch.Layer(detect.NPMDependency).ReadMetadata(&metadata)
						Expect(metadata.SHA256).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
					})
				})
			})
		})
	})
}
