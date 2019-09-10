package modules_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"

	"github.com/cloudfoundry/libcfbuildpack/layers"

	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/cloudfoundry/npm-cnb/modules"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

//go:generate mockgen -source=modules.go -destination=mocks_test.go -package=modules_test

func TestUnitModules(t *testing.T) {
	spec.Run(t, "Modules", testModules, spec.Report(report.Terminal{}))
}

func testModules(t *testing.T, when spec.G, it spec.S) {
	var (
		mockCtrl       *gomock.Controller
		mockPkgManager *MockPackageManager
		factory        *test.BuildFactory
	)

	it.Before(func() {
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)
		mockPkgManager = NewMockPackageManager(mockCtrl)

		factory = test.NewBuildFactory(t)
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("NewContributor", func() {
		it("returns true if a build plan exists with the dep", func() {
			factory.AddPlan(buildpackplan.Plan{Name: modules.Dependency})

			_, willContribute, err := modules.NewContributor(factory.Build, mockPkgManager)
			Expect(err).NotTo(HaveOccurred())
			Expect(willContribute).To(BeTrue())
		})

		it("returns false if a build plan does not exist with the dep", func() {
			_, willContribute, err := modules.NewContributor(factory.Build, mockPkgManager)
			Expect(err).NotTo(HaveOccurred())
			Expect(willContribute).To(BeFalse())
		})

		it("uses package-lock.json for identity when it exists", func() {
			test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "package-lock.json"), "package lock")
			factory.AddPlan(buildpackplan.Plan{Name: modules.Dependency})

			contributor, _, _ := modules.NewContributor(factory.Build, mockPkgManager)
			name, version := contributor.NodeModulesMetadata.Identity()
			Expect(name).To(Equal(modules.ModulesMetaName))
			Expect(version).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
		})
	})

	when("Contribute", func() {
		var (
			contributor                                      modules.Contributor
			willContribute                                   bool
			nodeModulesLayerRoot, npmCacheLayerRoot, appRoot string
			err                                              error
		)
		it.Before(func() {
			nodeModulesLayerRoot = factory.Build.Layers.Layer(modules.Dependency).Root
			npmCacheLayerRoot = factory.Build.Layers.Layer(modules.Cache).Root
			appRoot = factory.Build.Application.Root
			factory.AddPlan(buildpackplan.Plan{
				Name:     modules.Dependency,
				Metadata: buildpackplan.Metadata{"build": true, "launch": true},
			})

			contributor, willContribute, err = modules.NewContributor(factory.Build, mockPkgManager)
			Expect(err).NotTo(HaveOccurred())
			Expect(willContribute).To(BeTrue())

			mockPkgManager.EXPECT().WarnUnmetDependencies(appRoot)
		})
		when("the app is not vendored", func() {
			it.Before(func() {
				mockPkgManager.EXPECT().Install(nodeModulesLayerRoot, npmCacheLayerRoot, appRoot).Do(func(_, _, location string) {
					module := filepath.Join(location, modules.ModulesDir, "test_module")
					test.WriteFile(t, module, "some module")

					cache := filepath.Join(location, modules.CacheDir, "test_cache_item")
					test.WriteFile(t, cache, "some cache contents")
				})
			})

			it("installs node_modules, sets environment vars, writes layer metadata", func() {
				Expect(contributor.Contribute()).To(Succeed())

				nodeModulesLayer := factory.Build.Layers.Layer(modules.Dependency)
				Expect(nodeModulesLayer).To(test.HaveLayerMetadata(true, true, true))

				Expect(filepath.Join(nodeModulesLayer.Root, modules.ModulesDir, "test_module")).To(BeARegularFile())
				Expect(nodeModulesLayer).To(test.HaveOverrideSharedEnvironment("NODE_PATH", filepath.Join(nodeModulesLayer.Root, modules.ModulesDir)))
				Expect(nodeModulesLayer).To(test.HaveAppendPathSharedEnvironment("PATH", filepath.Join(nodeModulesLayer.Root, modules.ModulesDir, ".bin")))

				npmCacheLayer := factory.Build.Layers.Layer(modules.Cache)
				Expect(npmCacheLayer).To(test.HaveLayerMetadata(false, true, false))
				Expect(filepath.Join(npmCacheLayer.Root, modules.CacheDir, "test_cache_item")).To(BeARegularFile())

				Expect(filepath.Join(factory.Build.Application.Root, modules.ModulesDir)).NotTo(BeADirectory())
				Expect(filepath.Join(factory.Build.Application.Root, modules.CacheDir)).NotTo(BeADirectory())

				Expect(factory.Build.Layers).To(test.HaveApplicationMetadata(layers.Metadata{Processes: []layers.Process{{"web", "npm start", false}}}))

				nodeModulesMetadataName, _ := contributor.NodeModulesMetadata.Identity()
				npmCacheMetadataName, _ := contributor.NPMCacheMetadata.Identity()
				Expect(nodeModulesMetadataName).To(Equal(modules.ModulesMetaName))
				Expect(npmCacheMetadataName).To(Equal(modules.CacheMetaName))
			})

		})
		when("the app is vendored", func() {
			it.Before(func() {
				test.WriteFile(t, filepath.Join(factory.Build.Application.Root, modules.ModulesDir, "test_module"), "some module")
				mockPkgManager.EXPECT().Rebuild(gomock.Any(), factory.Build.Application.Root)
			})

			it("rebuilds the node_modules and moves them out of the app dir", func() {
				Expect(contributor.Contribute()).To(Succeed())

				layer := factory.Build.Layers.Layer(modules.Dependency)
				Expect(filepath.Join(layer.Root, modules.ModulesDir, "test_module")).To(BeARegularFile())
				Expect(filepath.Join(factory.Build.Application.Root, modules.ModulesDir)).NotTo(BeADirectory())
			})
		})
	})
}
