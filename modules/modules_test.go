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
				test.WriteFile(
					t,
					filepath.Join(factory.Build.Application.Root, "package-lock.json"),
					"package lock",
				)
			})

			it("returns true if a build plan exists", func() {
				factory.AddBuildPlan(modules.Dependency, buildplan.Dependency{})

				_, willContribute, err := modules.NewContributor(factory.Build, mockPkgManager)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
			})

			it("returns false if a build plan does not exist", func() {
				_, willContribute, err := modules.NewContributor(factory.Build, mockPkgManager)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeFalse())
			})

			it("uses package-lock.json for identity", func() {
				factory.AddBuildPlan(modules.Dependency, buildplan.Dependency{})

				contributor, _, _ := modules.NewContributor(factory.Build, mockPkgManager)
				name, version := contributor.Metadata.Identity()
				Expect(name).To(Equal(modules.Dependency))
				Expect(version).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
			})

			when("the app is vendored", func() {
				it.Before(func() {
					test.WriteFile(
						t,
						filepath.Join(factory.Build.Application.Root, "node_modules", "test_module"),
						"some module",
					)

					mockPkgManager.EXPECT().Rebuild(factory.Build.Application.Root)
				})

				it("contributes modules to the cache layer when included in the build plan", func() {
					factory.AddBuildPlan(modules.Dependency, buildplan.Dependency{
						Metadata: buildplan.Metadata{"build": true},
					})

					contributor, _, err := modules.NewContributor(factory.Build, mockPkgManager)
					Expect(err).NotTo(HaveOccurred())

					Expect(contributor.Contribute()).To(Succeed())

					layer := factory.Build.Layers.Layer(modules.Dependency)
					Expect(layer).To(test.HaveLayerMetadata(true, true, false))
					Expect(filepath.Join(layer.Root, "node_modules", "test_module")).To(BeARegularFile())
					Expect(layer).To(test.HaveOverrideSharedEnvironment("NODE_PATH", filepath.Join(layer.Root, "node_modules")))

					Expect(filepath.Join(factory.Build.Application.Root, "node_modules")).NotTo(BeADirectory())
				})

				it("contributes modules to the launch layer when included in the build plan", func() {
					factory.AddBuildPlan(modules.Dependency, buildplan.Dependency{
						Metadata: buildplan.Metadata{"launch": true},
					})

					contributor, _, err := modules.NewContributor(factory.Build, mockPkgManager)
					Expect(err).NotTo(HaveOccurred())

					Expect(contributor.Contribute()).To(Succeed())

					Expect(factory.Build.Layers).To(test.HaveLaunchMetadata(layers.Metadata{Processes: []layers.Process{{"web", "npm start"}}}))

					layer := factory.Build.Layers.Layer(modules.Dependency)
					Expect(layer).To(test.HaveLayerMetadata(false, true, true))
					Expect(filepath.Join(layer.Root, "node_modules", "test_module")).To(BeARegularFile())
					Expect(layer).To(test.HaveOverrideSharedEnvironment("NODE_PATH", filepath.Join(layer.Root, "node_modules")))

					Expect(filepath.Join(factory.Build.Application.Root, "node_modules")).NotTo(BeADirectory())
				})
			})

			when("the app is not vendored", func() {
				it.Before(func() {
					layerRoot := factory.Build.Layers.Layer(modules.Dependency).Root
					appRoot := factory.Build.Application.Root

					mockPkgManager.EXPECT().Install(layerRoot, appRoot).Do(func(_, location string) {
						module := filepath.Join(location, "node_modules", "test_module")
						test.WriteFile(t, module, "some module")

						cache := filepath.Join(location, "npm-cache", "test_cache_item")
						test.WriteFile(t, cache, "some cache contents")
					})
				})

				it("contributes modules to the cache layer when included in the build plan", func() {
					factory.AddBuildPlan(modules.Dependency, buildplan.Dependency{
						Metadata: buildplan.Metadata{"build": true},
					})

					contributor, _, err := modules.NewContributor(factory.Build, mockPkgManager)
					Expect(err).NotTo(HaveOccurred())

					Expect(contributor.Contribute()).To(Succeed())

					layer := factory.Build.Layers.Layer(modules.Dependency)
					Expect(layer).To(test.HaveLayerMetadata(true, true, false))
					Expect(filepath.Join(layer.Root, "node_modules", "test_module")).To(BeARegularFile())
					Expect(filepath.Join(layer.Root, "npm-cache", "test_cache_item")).To(BeARegularFile())
					Expect(layer).To(test.HaveOverrideSharedEnvironment("NODE_PATH", filepath.Join(layer.Root, "node_modules")))

					Expect(filepath.Join(factory.Build.Application.Root, "node_modules")).NotTo(BeADirectory())
					Expect(filepath.Join(factory.Build.Application.Root, "npm-cache")).NotTo(BeADirectory())
				})

				it("contributes modules to the launch layer when included in the build plan", func() {
					factory.AddBuildPlan(modules.Dependency, buildplan.Dependency{
						Metadata: buildplan.Metadata{"launch": true},
					})

					contributor, _, err := modules.NewContributor(factory.Build, mockPkgManager)
					Expect(err).NotTo(HaveOccurred())

					Expect(contributor.Contribute()).To(Succeed())

					Expect(factory.Build.Layers).To(test.HaveLaunchMetadata(layers.Metadata{Processes: []layers.Process{{"web", "npm start"}}}))

					layer := factory.Build.Layers.Layer(modules.Dependency)
					Expect(layer).To(test.HaveLayerMetadata(false, true, true))
					Expect(filepath.Join(layer.Root, "node_modules", "test_module")).To(BeARegularFile())
					Expect(filepath.Join(layer.Root, "npm-cache", "test_cache_item")).To(BeARegularFile())
					Expect(layer).To(test.HaveOverrideSharedEnvironment("NODE_PATH", filepath.Join(layer.Root, "node_modules")))

					Expect(filepath.Join(factory.Build.Application.Root, "node_modules")).NotTo(BeADirectory())
					Expect(filepath.Join(factory.Build.Application.Root, "npm-cache")).NotTo(BeADirectory())
				})
			})
		})
	})
}
