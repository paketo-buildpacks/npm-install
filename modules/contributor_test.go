package modules_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/cloudfoundry/npm-cnb/modules"
	"github.com/golang/mock/gomock"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=contributor.go -destination=mocks_test.go -package=modules_test

func testContributor(t *testing.T, when spec.G, it spec.S) {
	var (
		mockCtrl       *gomock.Controller
		mockPkgManager *MockPackageManager
		factory        *test.BuildFactory
		now            time.Time
	)

	it.Before(func() {
		RegisterTestingT(t)
		mockCtrl = gomock.NewController(t)
		mockPkgManager = NewMockPackageManager(mockCtrl)

		factory = test.NewBuildFactory(t)

		var err error
		now, err = time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		mockCtrl.Finish()
	})

	when("NewContributor", func() {
		when("a plan for the dependency exists", func() {
			it.Before(func() {
				factory.AddPlan(buildpackplan.Plan{Name: modules.Dependency})
			})

			it("returns true", func() {
				factory.AddPlan(buildpackplan.Plan{Name: modules.Dependency})

				_, willContribute, err := modules.NewContributor(factory.Build, mockPkgManager)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
			})
		})

		when("a plan for the dependency does NOT exist", func() {
			it("returns false if a build plan does not exist with the dep", func() {
				_, willContribute, err := modules.NewContributor(factory.Build, mockPkgManager)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeFalse())
			})
		})
	})

	when("Contribute", func() {
		var (
			contributor                     modules.Contributor
			willContribute                  bool
			nodeModulesLayer, npmCacheLayer layers.Layer
			appRoot                         string
			err                             error
		)

		it.Before(func() {
			nodeModulesLayer = factory.Build.Layers.Layer(modules.Dependency)
			npmCacheLayer = factory.Build.Layers.Layer(modules.Cache)
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

		when("the package-lock.json file exists", func() {
			it.Before(func() {
				packageLockPath := filepath.Join(appRoot, "package-lock.json")

				test.WriteFile(t, packageLockPath, "package lock")
				mockPkgManager.EXPECT().CI(gomock.Any(), gomock.Any(), gomock.Any())
			})

			it("uses package-lock.json for identity", func() {
				err := contributor.Contribute(now)
				Expect(err).NotTo(HaveOccurred())

				var metadata modules.Metadata
				err = nodeModulesLayer.ReadMetadata(&metadata)
				Expect(err).NotTo(HaveOccurred())

				Expect(metadata.Name).To(Equal(modules.ModulesMetaName))
				Expect(metadata.Hash).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))

				err = npmCacheLayer.ReadMetadata(&metadata)
				Expect(err).NotTo(HaveOccurred())

				Expect(metadata.Name).To(Equal(modules.CacheMetaName))
				Expect(metadata.Hash).To(Equal("152468741c83af08df4394d612172b58b2e7dca7164b5e6b79c5f6e96b829f77"))
			})
		})

		when("the package-lock.json file does NOT exist", func() {
			it.Before(func() {
				mockPkgManager.EXPECT().Install(gomock.Any(), gomock.Any(), gomock.Any())
			})

			it("uses current time for identity", func() {
				err := contributor.Contribute(now)
				Expect(err).NotTo(HaveOccurred())

				var metadata modules.Metadata
				err = nodeModulesLayer.ReadMetadata(&metadata)
				Expect(err).NotTo(HaveOccurred())

				Expect(metadata.Name).To(Equal(modules.ModulesMetaName))
				Expect(metadata.Hash).To(Equal("b3cf032436957e22a4c99036760a59c65bd4bdd981f3d2f02e4d3a80a0cf0cfe"))

				err = npmCacheLayer.ReadMetadata(&metadata)
				Expect(err).NotTo(HaveOccurred())

				Expect(metadata.Name).To(Equal(modules.CacheMetaName))
				Expect(metadata.Hash).To(Equal("b3cf032436957e22a4c99036760a59c65bd4bdd981f3d2f02e4d3a80a0cf0cfe"))
			})
		})

		when("testing the installation/rebuild combinations", func() {
			expectInstall := func() {
				mockPkgManager.EXPECT().Install(gomock.Any(), gomock.Any(), gomock.Any())
			}

			expectRebuild := func() {
				mockPkgManager.EXPECT().Rebuild(gomock.Any(), gomock.Any())
			}

			expectCI := func() {
				mockPkgManager.EXPECT().CI(gomock.Any(), gomock.Any(), gomock.Any())
			}

			for _, cfg := range []struct {
				Locked      bool
				Vendored    bool
				Cached      bool
				Expectation func()
			}{
				{Locked: false, Vendored: false, Cached: false, Expectation: expectInstall},
				{Locked: false, Vendored: false, Cached: true, Expectation: expectInstall},
				{Locked: false, Vendored: true, Cached: false, Expectation: expectRebuild},
				{Locked: false, Vendored: true, Cached: true, Expectation: expectRebuild},
				{Locked: true, Vendored: false, Cached: false, Expectation: expectCI},
				{Locked: true, Vendored: false, Cached: true, Expectation: expectCI},
				{Locked: true, Vendored: true, Cached: false, Expectation: expectRebuild},
				{Locked: true, Vendored: true, Cached: true, Expectation: expectCI},
			} {
				config := cfg // copy loop variable onto loop local
				when(fmt.Sprintf("the app is locked(%t), vendored(%t), and cached(%t)", config.Locked, config.Vendored, config.Cached), func() {
					it("runs the correct command", func() {
						if config.Locked {
							_, err := os.Create(filepath.Join(appRoot, modules.PackageLock))
							Expect(err).NotTo(HaveOccurred())
						}

						if config.Vendored {
							err := os.Mkdir(filepath.Join(appRoot, modules.ModulesDir), 0755)
							Expect(err).NotTo(HaveOccurred())
						}

						if config.Cached {
							err := os.Mkdir(filepath.Join(appRoot, modules.CacheDir), 0755)
							Expect(err).NotTo(HaveOccurred())
						}

						config.Expectation()

						Expect(contributor.Contribute(now)).To(Succeed())
					})
				})
			}
		})

		when("the app is not vendored", func() {
			it.Before(func() {
				mockPkgManager.EXPECT().Install(nodeModulesLayer.Root, npmCacheLayer.Root, appRoot).Do(func(_, _, location string) {
					module := filepath.Join(location, modules.ModulesDir, "test_module")
					test.WriteFile(t, module, "some module")

					cache := filepath.Join(location, modules.CacheDir, "test_cache_item")
					test.WriteFile(t, cache, "some cache contents")
				})
			})

			it("installs node_modules, sets environment vars, writes layer metadata", func() {
				Expect(contributor.Contribute(now)).To(Succeed())

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

				Expect(factory.Build.Layers).To(test.HaveApplicationMetadata(layers.Metadata{
					Processes: []layers.Process{
						{
							Type:    "web",
							Command: "npm start",
							Direct:  false,
						},
					},
				}))

				var metadata modules.Metadata
				err = nodeModulesLayer.ReadMetadata(&metadata)
				Expect(err).NotTo(HaveOccurred())
				Expect(metadata.Name).To(Equal(modules.ModulesMetaName))

				err = npmCacheLayer.ReadMetadata(&metadata)
				Expect(err).NotTo(HaveOccurred())
				Expect(metadata.Name).To(Equal(modules.CacheMetaName))
			})
		})

		when("the app is vendored", func() {
			it.Before(func() {
				test.WriteFile(t, filepath.Join(factory.Build.Application.Root, modules.ModulesDir, "test_module"), "some module")
				mockPkgManager.EXPECT().Rebuild(gomock.Any(), factory.Build.Application.Root)
			})

			it("rebuilds the node_modules and moves them out of the app dir", func() {
				Expect(contributor.Contribute(now)).To(Succeed())

				layer := factory.Build.Layers.Layer(modules.Dependency)
				Expect(filepath.Join(layer.Root, modules.ModulesDir, "test_module")).To(BeARegularFile())
				Expect(filepath.Join(factory.Build.Application.Root, modules.ModulesDir)).NotTo(BeADirectory())
			})
		})
	})
}
