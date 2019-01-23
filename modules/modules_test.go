package modules_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/layers"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/cloudfoundry/npm-cnb/modules"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

//go:generate mockgen -source=modules.go -destination=mocks_test.go -package=modules_test

func TestUnitModules(t *testing.T) {
	RegisterTestingT(t)
	spec.Run(t, "Modules", testModules, spec.Report(report.Terminal{}))
}

func testModules(t *testing.T, when spec.G, it spec.S) {
	when("modules.NewContributor", func() {
		var (
			mockCtrl       *gomock.Controller
			mockPkgManager *MockPackageManager
			factory        *test.BuildFactory
		)

		it.Before(func() {
			mockCtrl = gomock.NewController(t)
			mockPkgManager = NewMockPackageManager(mockCtrl)

			factory = test.NewBuildFactory(t)
		})

		it.After(func() {
			mockCtrl.Finish()
		})

		when("there is no package-lock.json", func() {
			it("fails", func() {
				factory.AddBuildPlan(modules.Dependency, buildplan.Dependency{})

				_, _, err := modules.NewContributor(factory.Build, mockPkgManager)
				Expect(err).To(HaveOccurred())
			})
		})

		when("there is a package-lock.json", func() {
			it.Before(func() {
				test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "package-lock.json"), "package lock")
			})

			it("returns true if a build plan exists with the dep", func() {
				factory.AddBuildPlan(modules.Dependency, buildplan.Dependency{})

				_, willContribute, err := modules.NewContributor(factory.Build, mockPkgManager)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
			})

			it("returns false if a build plan does not exist with the dep", func() {
				_, willContribute, err := modules.NewContributor(factory.Build, mockPkgManager)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeFalse())
			})

			it("uses package-lock.json for identity", func() {
				factory.AddBuildPlan(modules.Dependency, buildplan.Dependency{})

				contributor, _, _ := modules.NewContributor(factory.Build, mockPkgManager)
				name, version := contributor.NodeModulesMetadata.Identity()
				Expect(name).To(Equal(modules.Dependency))
				Expect(version).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
			})

			when("the app is vendored", func() {
				it.Before(func() {
					test.WriteFile(t, filepath.Join(factory.Build.Application.Root, modules.ModulesDir, "test_module"), "some module")
					mockPkgManager.EXPECT().Rebuild(factory.Build.Application.Root)
				})

				it("contributes for the build phase", func() {
					factory.AddBuildPlan(modules.Dependency, buildplan.Dependency{
						Metadata: buildplan.Metadata{"build": true},
					})

					contributor, _, err := modules.NewContributor(factory.Build, mockPkgManager)
					Expect(err).NotTo(HaveOccurred())

					Expect(contributor.Contribute()).To(Succeed())

					layer := factory.Build.Layers.Layer(modules.Dependency)
					Expect(layer).To(test.HaveLayerMetadata(true, true, false))
					Expect(filepath.Join(layer.Root, modules.ModulesDir, "test_module")).To(BeARegularFile())
					Expect(layer).To(test.HaveOverrideSharedEnvironment("NODE_PATH", filepath.Join(layer.Root, modules.ModulesDir)))

					Expect(filepath.Join(factory.Build.Application.Root, modules.ModulesDir)).NotTo(BeADirectory())
				})

				it("contributes for the launch phase", func() {
					factory.AddBuildPlan(modules.Dependency, buildplan.Dependency{
						Metadata: buildplan.Metadata{"launch": true},
					})

					contributor, _, err := modules.NewContributor(factory.Build, mockPkgManager)
					Expect(err).NotTo(HaveOccurred())

					Expect(contributor.Contribute()).To(Succeed())

					Expect(factory.Build.Layers).To(test.HaveLaunchMetadata(layers.Metadata{Processes: []layers.Process{{"web", "npm start"}}}))

					layer := factory.Build.Layers.Layer(modules.Dependency)
					Expect(layer).To(test.HaveLayerMetadata(false, true, true))
					Expect(filepath.Join(layer.Root, modules.ModulesDir, "test_module")).To(BeARegularFile())
					Expect(layer).To(test.HaveOverrideSharedEnvironment("NODE_PATH", filepath.Join(layer.Root, modules.ModulesDir)))

					Expect(filepath.Join(factory.Build.Application.Root, modules.ModulesDir)).NotTo(BeADirectory())
				})
			})

			when("the app is not vendored", func() {
				it.Before(func() {
					nodeModulesLayerRoot := factory.Build.Layers.Layer(modules.Dependency).Root
					npmCacheLayerRoot := factory.Build.Layers.Layer(modules.Cache).Root
					appRoot := factory.Build.Application.Root

					mockPkgManager.EXPECT().Install(nodeModulesLayerRoot, npmCacheLayerRoot, appRoot).Do(func(_, _, location string) {
						module := filepath.Join(location, modules.ModulesDir, "test_module")
						test.WriteFile(t, module, "some module")

						cache := filepath.Join(location, modules.CacheDir, "test_cache_item")
						test.WriteFile(t, cache, "some cache contents")
					})
				})

				it("contributes for the build phase", func() {
					factory.AddBuildPlan(modules.Dependency, buildplan.Dependency{
						Metadata: buildplan.Metadata{"build": true},
					})

					contributor, _, err := modules.NewContributor(factory.Build, mockPkgManager)
					Expect(err).NotTo(HaveOccurred())

					Expect(contributor.Contribute()).To(Succeed())

					nodeModulesLayer := factory.Build.Layers.Layer(modules.Dependency)
					Expect(nodeModulesLayer).To(test.HaveLayerMetadata(true, true, false))
					Expect(filepath.Join(nodeModulesLayer.Root, modules.ModulesDir, "test_module")).To(BeARegularFile())
					Expect(nodeModulesLayer).To(test.HaveOverrideSharedEnvironment("NODE_PATH", filepath.Join(nodeModulesLayer.Root, modules.ModulesDir)))

					npmCacheLayer := factory.Build.Layers.Layer(modules.Cache)
					Expect(npmCacheLayer).To(test.HaveLayerMetadata(false, true, false))
					Expect(filepath.Join(npmCacheLayer.Root, modules.CacheDir, "test_cache_item")).To(BeARegularFile())

					Expect(filepath.Join(factory.Build.Application.Root, modules.ModulesDir)).NotTo(BeADirectory())
					Expect(filepath.Join(factory.Build.Application.Root, modules.CacheDir)).NotTo(BeADirectory())
				})

				it("contributes for the launch phase", func() {
					factory.AddBuildPlan(modules.Dependency, buildplan.Dependency{
						Metadata: buildplan.Metadata{"launch": true},
					})

					contributor, _, err := modules.NewContributor(factory.Build, mockPkgManager)
					Expect(err).NotTo(HaveOccurred())

					Expect(contributor.Contribute()).To(Succeed())

					Expect(factory.Build.Layers).To(test.HaveLaunchMetadata(layers.Metadata{Processes: []layers.Process{{"web", "npm start"}}}))

					nodeModulesLayer := factory.Build.Layers.Layer(modules.Dependency)
					Expect(nodeModulesLayer).To(test.HaveLayerMetadata(false, true, true))
					Expect(filepath.Join(nodeModulesLayer.Root, modules.ModulesDir, "test_module")).To(BeARegularFile())
					Expect(nodeModulesLayer).To(test.HaveOverrideSharedEnvironment("NODE_PATH", filepath.Join(nodeModulesLayer.Root, modules.ModulesDir)))

					npmCacheLayer := factory.Build.Layers.Layer(modules.Cache)
					Expect(npmCacheLayer).To(test.HaveLayerMetadata(false, true, false))
					Expect(filepath.Join(npmCacheLayer.Root, modules.CacheDir, "test_cache_item")).To(BeARegularFile())

					Expect(filepath.Join(factory.Build.Application.Root, modules.ModulesDir)).NotTo(BeADirectory())
					Expect(filepath.Join(factory.Build.Application.Root, modules.CacheDir)).NotTo(BeADirectory())
				})
			})
		})
	})
}
