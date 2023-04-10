package npminstall_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	npminstall "github.com/paketo-buildpacks/npm-install"
	"github.com/paketo-buildpacks/npm-install/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"

	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		workingDir string
		cnbDir     string

		processLayerDir   string
		processWorkingDir string
		processCacheDir   string
		processNpmrcPath  string

		projectPathParser    *fakes.PathParser
		buildProcess         *fakes.BuildProcess
		buildManager         *fakes.BuildManager
		configurationManager *fakes.ConfigurationManager
		entryResolver        *fakes.EntryResolver
		pruneProcess         *fakes.PruneProcess
		sbomGenerator        *fakes.SBOMGenerator
		linker               *fakes.Symlinker
		environment          *fakes.EnvironmentConfig
		symlinkResolver      *fakes.SymlinkResolver

		buffer *bytes.Buffer

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		projectPathParser = &fakes.PathParser{}
		projectPathParser.GetCall.Returns.ProjectPath = ""

		buildProcess = &fakes.BuildProcess{}
		buildProcess.ShouldRunCall.Returns.Run = true
		buildProcess.ShouldRunCall.Returns.Sha = "some-sha"
		buildProcess.RunCall.Stub = func(ld, cd, wd, rc string, l bool) error {
			err := os.MkdirAll(filepath.Join(ld, "node_modules"), os.ModePerm)
			if err != nil {
				return err
			}

			err = os.MkdirAll(filepath.Join(cd, "layer-content"), os.ModePerm)
			if err != nil {
				return err
			}
			processLayerDir = ld
			processCacheDir = cd
			processWorkingDir = wd
			processNpmrcPath = rc

			return nil
		}

		buildManager = &fakes.BuildManager{}
		buildManager.ResolveCall.Returns.BuildProcess = buildProcess

		configurationManager = &fakes.ConfigurationManager{}

		entryResolver = &fakes.EntryResolver{}

		buffer = bytes.NewBuffer(nil)
		logger := scribe.NewEmitter(buffer)

		pruneProcess = &fakes.PruneProcess{}

		sbomGenerator = &fakes.SBOMGenerator{}
		sbomGenerator.GenerateCall.Returns.SBOM = sbom.SBOM{}

		linker = &fakes.Symlinker{}

		environment = &fakes.EnvironmentConfig{}
		environment.LookupBoolCall.Returns.Bool = false

		symlinkResolver = &fakes.SymlinkResolver{}

		build = npminstall.Build(
			projectPathParser,
			entryResolver,
			configurationManager,
			buildManager,
			pruneProcess,
			chronos.DefaultClock,
			logger,
			sbomGenerator,
			linker,
			environment,
			symlinkResolver,
		)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
	})

	context("when required during build", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Build = true
		})

		it("returns a result that installs build modules", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					SBOMFormats: []string{"application/vnd.cyclonedx+json", "application/spdx+json", "application/vnd.syft+json"},
				},
				Platform: packit.Platform{
					Path: "some-platform-path",
				},
				WorkingDir: workingDir,
				CNBPath:    cnbDir,
				Layers:     packit.Layers{Path: layersDir},
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(len(result.Layers)).To(Equal(2))

			buildLayer := result.Layers[0]
			Expect(buildLayer.Name).To(Equal("build-modules"))
			Expect(buildLayer.Path).To(Equal(filepath.Join(layersDir, "build-modules")))
			Expect(buildLayer.SharedEnv).To(Equal(packit.Environment{}))
			Expect(buildLayer.BuildEnv).To(Equal(packit.Environment{
				"PATH.append":       filepath.Join(layersDir, "build-modules", "node_modules", ".bin"),
				"PATH.delim":        ":",
				"NODE_ENV.override": "development",
			}))
			Expect(buildLayer.LaunchEnv).To(Equal(packit.Environment{}))
			Expect(buildLayer.ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(buildLayer.Build).To(BeTrue())
			Expect(buildLayer.Launch).To(BeFalse())
			Expect(buildLayer.Cache).To(BeTrue())
			Expect(buildLayer.Metadata).To(Equal(map[string]interface{}{
				"cache_sha": "some-sha",
			}))

			Expect(buildLayer.SBOM.Formats()).To(HaveLen(3))

			cdx := buildLayer.SBOM.Formats()[0]
			spdx := buildLayer.SBOM.Formats()[1]
			syft := buildLayer.SBOM.Formats()[2]

			Expect(cdx.Extension).To(Equal("cdx.json"))
			Expect(spdx.Extension).To(Equal("spdx.json"))
			Expect(syft.Extension).To(Equal("syft.json"))

			content, err := io.ReadAll(cdx.Content)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(MatchJSON(`{
				"bomFormat": "CycloneDX",
				"components": [],
				"metadata": {
					"tools": [
						{
							"name": "syft",
							"vendor": "anchore",
							"version": "[not provided]"
						}
					]
				},
				"specVersion": "1.3",
				"version": 1
			}`))

			content, err = io.ReadAll(spdx.Content)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(MatchJSON(`{
				"SPDXID": "SPDXRef-DOCUMENT",
				"creationInfo": {
					"created": "0001-01-01T00:00:00Z",
					"creators": [
						"Organization: Anchore, Inc",
						"Tool: syft-"
					],
					"licenseListVersion": "3.16"
				},
				"dataLicense": "CC0-1.0",
				"documentNamespace": "https://paketo.io/packit/unknown-source-type/unknown-88cfa225-65e0-5755-895f-c1c8f10fde76",
				"name": "unknown",
				"relationships": [
					{
						"relatedSpdxElement": "SPDXRef-DOCUMENT",
						"relationshipType": "DESCRIBES",
						"spdxElementId": "SPDXRef-DOCUMENT"
					}
				],
				"spdxVersion": "SPDX-2.2"
			}`))

			content, err = io.ReadAll(syft.Content)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(MatchJSON(`{
				"artifacts": [],
				"artifactRelationships": [],
				"source": {
					"type": "",
					"target": null
				},
				"distro": {},
				"descriptor": {
					"name": "",
					"version": ""
				},
				"schema": {
					"version": "3.0.1",
					"url": "https://raw.githubusercontent.com/anchore/syft/main/schema/json/schema-3.0.1.json"
				}
			}`))

			cacheLayer := result.Layers[1]
			Expect(cacheLayer.Name).To(Equal(npminstall.LayerNameCache))
			Expect(cacheLayer.Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
			Expect(cacheLayer.SharedEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.BuildEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.LaunchEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(cacheLayer.Build).To(BeFalse())
			Expect(cacheLayer.Launch).To(BeFalse())
			Expect(cacheLayer.Cache).To(BeTrue())

			Expect(projectPathParser.GetCall.Receives.Path).To(Equal(workingDir))

			Expect(configurationManager.DeterminePathCall.Receives.Typ).To(Equal("npmrc"))
			Expect(configurationManager.DeterminePathCall.Receives.PlatformDir).To(Equal("some-platform-path"))
			Expect(configurationManager.DeterminePathCall.Receives.Entry).To(Equal(".npmrc"))

			Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(workingDir))

			Expect(processLayerDir).To(Equal(filepath.Join(layersDir, "build-modules")))
			Expect(processCacheDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
			Expect(processWorkingDir).To(Equal(workingDir))
			Expect(processNpmrcPath).To(Equal(""))

			Expect(linker.LinkCall.Receives.Source).To(Equal(filepath.Join(workingDir, "node_modules")))
			Expect(linker.LinkCall.Receives.Target).To(Equal(filepath.Join(layersDir, "build-modules", "node_modules")))

			Expect(symlinkResolver.ResolveCall.Receives.LockfilePath).To(Equal(filepath.Join(workingDir, "package-lock.json")))
			Expect(symlinkResolver.ResolveCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "build-modules")))
		})
	})

	context("when required during launch", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
		})

		it("returns a result that installs build modules", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					SBOMFormats: []string{"application/vnd.cyclonedx+json", "application/spdx+json", "application/vnd.syft+json"},
				},
				Platform: packit.Platform{
					Path: "some-platform-path",
				},
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(len(result.Layers)).To(Equal(2))

			launchLayer := result.Layers[0]
			Expect(launchLayer.Name).To(Equal("launch-modules"))
			Expect(launchLayer.Path).To(Equal(filepath.Join(layersDir, "launch-modules")))
			Expect(launchLayer.SharedEnv).To(Equal(packit.Environment{}))
			Expect(launchLayer.BuildEnv).To(Equal(packit.Environment{}))
			Expect(launchLayer.LaunchEnv).To(Equal(packit.Environment{
				"NPM_CONFIG_LOGLEVEL.default": "error",
				"NODE_PROJECT_PATH.default":   workingDir,
				"PATH.append":                 filepath.Join(layersDir, "launch-modules", "node_modules", ".bin"),
				"PATH.delim":                  ":",
			}))
			Expect(launchLayer.ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(launchLayer.Build).To(BeFalse())
			Expect(launchLayer.Launch).To(BeTrue())
			Expect(launchLayer.Cache).To(BeFalse())
			Expect(launchLayer.Metadata).To(Equal(
				map[string]interface{}{
					"cache_sha": "some-sha",
				}))

			Expect(launchLayer.SBOM.Formats()).To(HaveLen(3))

			cdx := launchLayer.SBOM.Formats()[0]
			spdx := launchLayer.SBOM.Formats()[1]
			syft := launchLayer.SBOM.Formats()[2]

			Expect(cdx.Extension).To(Equal("cdx.json"))
			Expect(spdx.Extension).To(Equal("spdx.json"))
			Expect(syft.Extension).To(Equal("syft.json"))

			content, err := io.ReadAll(cdx.Content)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(MatchJSON(`{
				"bomFormat": "CycloneDX",
				"components": [],
				"metadata": {
					"tools": [
						{
							"name": "syft",
							"vendor": "anchore",
							"version": "[not provided]"
						}
					]
				},
				"specVersion": "1.3",
				"version": 1
			}`))

			content, err = io.ReadAll(spdx.Content)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(MatchJSON(`{
				"SPDXID": "SPDXRef-DOCUMENT",
				"creationInfo": {
					"created": "0001-01-01T00:00:00Z",
					"creators": [
						"Organization: Anchore, Inc",
						"Tool: syft-"
					],
					"licenseListVersion": "3.16"
				},
				"dataLicense": "CC0-1.0",
				"documentNamespace": "https://paketo.io/packit/unknown-source-type/unknown-88cfa225-65e0-5755-895f-c1c8f10fde76",
				"name": "unknown",
				"relationships": [
					{
						"relatedSpdxElement": "SPDXRef-DOCUMENT",
						"relationshipType": "DESCRIBES",
						"spdxElementId": "SPDXRef-DOCUMENT"
					}
				],
				"spdxVersion": "SPDX-2.2"
			}`))

			content, err = io.ReadAll(syft.Content)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(MatchJSON(`{
				"artifacts": [],
				"artifactRelationships": [],
				"source": {
					"type": "",
					"target": null
				},
				"distro": {},
				"descriptor": {
					"name": "",
					"version": ""
				},
				"schema": {
					"version": "3.0.1",
					"url": "https://raw.githubusercontent.com/anchore/syft/main/schema/json/schema-3.0.1.json"
				}
			}`))

			cacheLayer := result.Layers[1]
			Expect(cacheLayer.Name).To(Equal(npminstall.LayerNameCache))
			Expect(cacheLayer.Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
			Expect(cacheLayer.SharedEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.BuildEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.LaunchEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(cacheLayer.Build).To(BeFalse())
			Expect(cacheLayer.Launch).To(BeFalse())
			Expect(cacheLayer.Cache).To(BeTrue())

			Expect(projectPathParser.GetCall.Receives.Path).To(Equal(workingDir))

			Expect(configurationManager.DeterminePathCall.Receives.Typ).To(Equal("npmrc"))
			Expect(configurationManager.DeterminePathCall.Receives.PlatformDir).To(Equal("some-platform-path"))
			Expect(configurationManager.DeterminePathCall.Receives.Entry).To(Equal(".npmrc"))

			Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(workingDir))

			Expect(pruneProcess.RunCall.CallCount).To(Equal(0))

			Expect(processLayerDir).To(Equal(filepath.Join(layersDir, "launch-modules")))
			Expect(processCacheDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
			Expect(processWorkingDir).To(Equal(workingDir))
			Expect(processNpmrcPath).To(Equal(""))

			Expect(linker.LinkCall.Receives.Source).To(Equal(filepath.Join(workingDir, "node_modules")))
			Expect(linker.LinkCall.Receives.Target).To(Equal(filepath.Join(layersDir, "launch-modules", "node_modules")))

			Expect(symlinkResolver.ResolveCall.Receives.LockfilePath).To(Equal(filepath.Join(workingDir, "package-lock.json")))
			Expect(symlinkResolver.ResolveCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "launch-modules")))
		})
	})

	context("when node_modules is required at build and launch", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			entryResolver.MergeLayerTypesCall.Returns.Build = true
		})

		it("resolves and calls the build process", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					SBOMFormats: []string{"application/vnd.cyclonedx+json", "application/spdx+json", "application/vnd.syft+json"},
				},
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "node_modules",
							Metadata: map[string]interface{}{
								"build":  true,
								"launch": true,
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(len(result.Layers)).To(Equal(3))

			buildLayer := result.Layers[0]
			Expect(buildLayer.Name).To(Equal("build-modules"))
			Expect(buildLayer.Path).To(Equal(filepath.Join(layersDir, "build-modules")))
			Expect(buildLayer.SharedEnv).To(Equal(packit.Environment{}))
			Expect(buildLayer.BuildEnv).To(Equal(packit.Environment{
				"PATH.append":       filepath.Join(layersDir, "build-modules", "node_modules", ".bin"),
				"PATH.delim":        ":",
				"NODE_ENV.override": "development",
			}))
			Expect(buildLayer.LaunchEnv).To(Equal(packit.Environment{}))
			Expect(buildLayer.ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(buildLayer.Build).To(BeTrue())
			Expect(buildLayer.Launch).To(BeFalse())
			Expect(buildLayer.Cache).To(BeTrue())
			Expect(buildLayer.Metadata).To(Equal(
				map[string]interface{}{
					"cache_sha": "some-sha",
				}))

			Expect(buildLayer.SBOM.Formats()).To(HaveLen(3))

			cdx := buildLayer.SBOM.Formats()[0]
			spdx := buildLayer.SBOM.Formats()[1]
			syft := buildLayer.SBOM.Formats()[2]

			Expect(cdx.Extension).To(Equal("cdx.json"))
			Expect(spdx.Extension).To(Equal("spdx.json"))
			Expect(syft.Extension).To(Equal("syft.json"))

			content, err := io.ReadAll(cdx.Content)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(MatchJSON(`{
				"bomFormat": "CycloneDX",
				"components": [],
				"metadata": {
					"tools": [
						{
							"name": "syft",
							"vendor": "anchore",
							"version": "[not provided]"
						}
					]
				},
				"specVersion": "1.3",
				"version": 1
			}`))

			content, err = io.ReadAll(spdx.Content)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(MatchJSON(`{
				"SPDXID": "SPDXRef-DOCUMENT",
				"creationInfo": {
					"created": "0001-01-01T00:00:00Z",
					"creators": [
						"Organization: Anchore, Inc",
						"Tool: syft-"
					],
					"licenseListVersion": "3.16"
				},
				"dataLicense": "CC0-1.0",
				"documentNamespace": "https://paketo.io/packit/unknown-source-type/unknown-88cfa225-65e0-5755-895f-c1c8f10fde76",
				"name": "unknown",
				"relationships": [
					{
						"relatedSpdxElement": "SPDXRef-DOCUMENT",
						"relationshipType": "DESCRIBES",
						"spdxElementId": "SPDXRef-DOCUMENT"
					}
				],
				"spdxVersion": "SPDX-2.2"
			}`))

			content, err = io.ReadAll(syft.Content)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(MatchJSON(`{
				"artifacts": [],
				"artifactRelationships": [],
				"source": {
					"type": "",
					"target": null
				},
				"distro": {},
				"descriptor": {
					"name": "",
					"version": ""
				},
				"schema": {
					"version": "3.0.1",
					"url": "https://raw.githubusercontent.com/anchore/syft/main/schema/json/schema-3.0.1.json"
				}
			}`))

			launchLayer := result.Layers[1]
			Expect(launchLayer.Name).To(Equal("launch-modules"))
			Expect(launchLayer.Path).To(Equal(filepath.Join(layersDir, "launch-modules")))
			Expect(launchLayer.SharedEnv).To(Equal(packit.Environment{}))
			Expect(launchLayer.BuildEnv).To(Equal(packit.Environment{}))
			Expect(launchLayer.LaunchEnv).To(Equal(packit.Environment{
				"NPM_CONFIG_LOGLEVEL.default": "error",
				"NODE_PROJECT_PATH.default":   workingDir,
				"PATH.append":                 filepath.Join(layersDir, "launch-modules", "node_modules", ".bin"),
				"PATH.delim":                  ":",
			}))
			Expect(launchLayer.ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(launchLayer.Build).To(BeFalse())
			Expect(launchLayer.Launch).To(BeTrue())
			Expect(launchLayer.Cache).To(BeFalse())
			Expect(launchLayer.Metadata).To(Equal(
				map[string]interface{}{
					"cache_sha": "some-sha",
				}))
			Expect(launchLayer.ExecD).To(Equal([]string{
				filepath.Join(cnbDir, "bin", "setup-symlinks"),
			}))

			Expect(launchLayer.SBOM.Formats()).To(HaveLen(3))

			cdx = launchLayer.SBOM.Formats()[0]
			spdx = launchLayer.SBOM.Formats()[1]
			syft = launchLayer.SBOM.Formats()[2]

			Expect(cdx.Extension).To(Equal("cdx.json"))
			Expect(spdx.Extension).To(Equal("spdx.json"))
			Expect(syft.Extension).To(Equal("syft.json"))

			content, err = io.ReadAll(cdx.Content)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(MatchJSON(`{
				"bomFormat": "CycloneDX",
				"components": [],
				"metadata": {
					"tools": [
						{
							"name": "syft",
							"vendor": "anchore",
							"version": "[not provided]"
						}
					]
				},
				"specVersion": "1.3",
				"version": 1
			}`))

			content, err = io.ReadAll(spdx.Content)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(MatchJSON(`{
				"SPDXID": "SPDXRef-DOCUMENT",
				"creationInfo": {
					"created": "0001-01-01T00:00:00Z",
					"creators": [
						"Organization: Anchore, Inc",
						"Tool: syft-"
					],
					"licenseListVersion": "3.16"
				},
				"dataLicense": "CC0-1.0",
				"documentNamespace": "https://paketo.io/packit/unknown-source-type/unknown-88cfa225-65e0-5755-895f-c1c8f10fde76",
				"name": "unknown",
				"relationships": [
					{
						"relatedSpdxElement": "SPDXRef-DOCUMENT",
						"relationshipType": "DESCRIBES",
						"spdxElementId": "SPDXRef-DOCUMENT"
					}
				],
				"spdxVersion": "SPDX-2.2"
			}`))

			content, err = io.ReadAll(syft.Content)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(MatchJSON(`{
				"artifacts": [],
				"artifactRelationships": [],
				"source": {
					"type": "",
					"target": null
				},
				"distro": {},
				"descriptor": {
					"name": "",
					"version": ""
				},
				"schema": {
					"version": "3.0.1",
					"url": "https://raw.githubusercontent.com/anchore/syft/main/schema/json/schema-3.0.1.json"
				}
			}`))

			cacheLayer := result.Layers[2]
			Expect(cacheLayer.Name).To(Equal(npminstall.LayerNameCache))
			Expect(cacheLayer.Path).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
			Expect(cacheLayer.SharedEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.BuildEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.LaunchEnv).To(Equal(packit.Environment{}))
			Expect(cacheLayer.ProcessLaunchEnv).To(Equal(map[string]packit.Environment{}))
			Expect(cacheLayer.Build).To(BeFalse())
			Expect(cacheLayer.Launch).To(BeFalse())
			Expect(cacheLayer.Cache).To(BeTrue())

			Expect(pruneProcess.RunCall.Receives.ModulesDir).To(Equal(launchLayer.Path))
			Expect(pruneProcess.RunCall.Receives.CacheDir).To(Equal(cacheLayer.Path))
			Expect(pruneProcess.RunCall.Receives.WorkingDir).To(Equal(workingDir))
			Expect(pruneProcess.RunCall.Receives.NpmrcPath).To(Equal(""))

			Expect(linker.LinkCall.Receives.Source).To(Equal(filepath.Join(workingDir, "node_modules")))
			Expect(linker.LinkCall.Receives.Target).To(Equal(filepath.Join(buildLayer.Path, "node_modules")))

			Expect(symlinkResolver.ResolveCall.Receives.LockfilePath).To(Equal(filepath.Join(workingDir, "package-lock.json")))
			Expect(symlinkResolver.ResolveCall.Receives.LayerPath).To(Equal(filepath.Join(buildLayer.Path)))
		})
	})

	context("when one npmrc binding is detected", func() {
		it.Before(func() {
			configurationManager.DeterminePathCall.Returns.Path = "some-binding-path/.npmrc"
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
		})

		it("passes the path to the .npmrc to the build process and env configuration", func() {
			_, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(buildProcess.ShouldRunCall.Receives.NpmrcPath).To(Equal("some-binding-path/.npmrc"))
			Expect(buildProcess.RunCall.Receives.NpmrcPath).To(Equal("some-binding-path/.npmrc"))
		})
	})

	context("when the build process should not run", func() {
		it.Before(func() {
			buildProcess.ShouldRunCall.Returns.Run = false
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			Expect(os.MkdirAll(filepath.Join(workingDir, "node_modules"), os.ModePerm))
		})

		it("resolves and skips build process", func() {
			_, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(projectPathParser.GetCall.Receives.Path).To(Equal(workingDir))
			Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(workingDir))
			Expect(buildProcess.RunCall.CallCount).To(Equal(0))

			Expect(linker.LinkCall.Receives.Source).To(Equal(filepath.Join(workingDir, "node_modules")))
			Expect(linker.LinkCall.Receives.Target).To(Equal(filepath.Join(layersDir, "launch-modules", "node_modules")))
		})

		context("when BP_NODE_PROJECT_PATH is set", func() {
			it.Before(func() {
				buildProcess.ShouldRunCall.Returns.Run = true
				projectPathParser.GetCall.Returns.ProjectPath = "some-dir"
				Expect(os.MkdirAll(filepath.Join(workingDir, "some-dir", "node_modules"), os.ModePerm))
			})

			it("resolves and calls the build process", func() {
				_, err := build(packit.BuildContext{
					BuildpackInfo: packit.BuildpackInfo{
						SBOMFormats: []string{"application/vnd.cyclonedx+json", "application/spdx+json", "application/vnd.syft+json"},
					},
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).NotTo(HaveOccurred())

				Expect(projectPathParser.GetCall.Receives.Path).To(Equal(workingDir))

				Expect(buildManager.ResolveCall.Receives.WorkingDir).To(Equal(filepath.Join(workingDir, "some-dir")))

				Expect(processLayerDir).To(Equal(filepath.Join(layersDir, "launch-modules")))
				Expect(processCacheDir).To(Equal(filepath.Join(layersDir, npminstall.LayerNameCache)))
				Expect(processWorkingDir).To(Equal(filepath.Join(workingDir, "some-dir")))
			})
		})

	})

	context("when the cache layer directory is empty", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Launch = true
			buildProcess.RunCall.Stub = func(ld, cd, wd, rc string, l bool) error {
				err := os.MkdirAll(cd, os.ModePerm)
				if err != nil {
					return err
				}

				return nil
			}
		})

		it("filters out empty layers", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(len(result.Layers)).To(Equal(1))
			Expect(result.Layers[0].Name).To(Equal("launch-modules"))
		})
	})

	context("when the cache layer directory does not exist", func() {
		it("filters out empty layers", func() {
			result, err := build(packit.BuildContext{
				WorkingDir: workingDir,
				Layers:     packit.Layers{Path: layersDir},
				CNBPath:    cnbDir,
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{Name: "node_modules"},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(len(result.Layers)).To(Equal(0))
		})
	})

	context("failure cases", func() {
		context("when the npm-cache layer cannot be fetched", func() {
			it.Before(func() {
				_, err := os.Create(filepath.Join(layersDir, "npm-cache.toml"))
				Expect(err).NotTo(HaveOccurred())
				Expect(os.Chmod(filepath.Join(layersDir, "npm-cache.toml"), 0000)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the configuration manager fails while determining the path", func() {
			it.Before(func() {
				configurationManager.DeterminePathCall.Returns.Err = errors.New("failed to determine path")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).To(MatchError("failed to determine path"))
			})
		})

		context("when the project path parser provided fails", func() {
			it.Before(func() {
				projectPathParser.GetCall.Returns.Err = errors.New("failed to parse project path")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).To(MatchError("failed to parse project path"))
			})
		})

		context("when the build process cannot be resolved", func() {
			it.Before(func() {
				buildManager.ResolveCall.Returns.Error = errors.New("failed to resolve build process")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					WorkingDir: workingDir,
					Layers:     packit.Layers{Path: layersDir},
					CNBPath:    cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "node_modules"},
						},
					},
				})
				Expect(err).To(MatchError("failed to resolve build process"))
			})
		})

		context("during the build installation process", func() {
			it.Before(func() {
				entryResolver.MergeLayerTypesCall.Returns.Build = true
			})

			context("when the node_modules layer cannot be fetched", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(layersDir, "build-modules.toml"), nil, 0000)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the build process cache check fails", func() {
				it.Before(func() {
					buildProcess.ShouldRunCall.Returns.Err = errors.New("failed to check cache")
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError("failed to check cache"))
				})
			})

			context("when the node_modules layer cannot be reset", func() {
				it.Before(func() {
					Expect(os.Chmod(layersDir, 4444)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(layersDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the build process provided fails", func() {
				it.Before(func() {
					buildProcess.RunCall.Stub = func(string, string, string, string, bool) error {
						return errors.New("given build process failed")
					}
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError("given build process failed"))
				})
			})

			context("when the BOM cannot be generated", func() {
				it.Before(func() {
					sbomGenerator.GenerateCall.Returns.Error = errors.New("failed to generate SBOM")
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						BuildpackInfo: packit.BuildpackInfo{
							SBOMFormats: []string{"application/vnd.cyclonedx+json", "application/spdx+json", "application/vnd.syft+json"},
						},
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{{Name: "node_modules"}},
						},
						Stack: "some-stack",
					})
					Expect(err).To(MatchError("failed to generate SBOM"))
				})
			})

			context("when the BOM cannot be formatted", func() {
				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						WorkingDir: workingDir,
						BuildpackInfo: packit.BuildpackInfo{
							SBOMFormats: []string{"random-format"},
						},
					})
					Expect(err).To(MatchError("unsupported SBOM format: 'random-format'"))
				})
			})

			context("when BP_DISABLE_SBOM is set incorrectly", func() {
				it.Before((func() {
					environment.LookupBoolCall.Returns.Error = errors.New("failed to parse bool")
				}))

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						Layers:  packit.Layers{Path: layersDir},
						CNBPath: cnbDir,
						BuildpackInfo: packit.BuildpackInfo{
							SBOMFormats: []string{"application/vnd.cyclonedx+json"},
						},
					})
					Expect(err).To(MatchError(ContainSubstring("failed to parse bool")))
				})
			})

			context("when the node_modules directory cannot be linked", func() {
				it.Before(func() {
					linker.LinkCall.Returns.Error = errors.New("failed to link node_modules")
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError("failed to link node_modules"))
				})
			})

			context("when the linked modules cannot be resolved", func() {
				it.Before(func() {
					symlinkResolver.ResolveCall.Returns.Error = errors.New("failed to resolve module symlinks")
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError("failed to resolve module symlinks"))
				})
			})
		})

		context("during the launch installation process", func() {
			it.Before(func() {
				entryResolver.MergeLayerTypesCall.Returns.Launch = true
			})

			context("when the node_modules layer cannot be fetched", func() {
				it.Before(func() {
					Expect(os.WriteFile(filepath.Join(layersDir, "launch-modules.toml"), nil, 0000)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the build process cache check fails", func() {
				it.Before(func() {
					buildProcess.ShouldRunCall.Returns.Err = errors.New("failed to check cache")
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError("failed to check cache"))
				})
			})

			context("when the node_modules layer cannot be reset", func() {
				it.Before(func() {
					Expect(os.Chmod(layersDir, 4444)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(layersDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when build is also set and the node_modules copy fails", func() {
				it.Before(func() {
					entryResolver.MergeLayerTypesCall.Returns.Build = true
					buildProcess.RunCall.Stub = func(string, string, string, string, bool) error {
						return nil
					}
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
				})
			})

			context("when the build process provided fails", func() {
				it.Before(func() {
					buildProcess.RunCall.Stub = func(string, string, string, string, bool) error {
						return errors.New("given build process failed")
					}
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError("given build process failed"))
				})
			})

			context("when the build process provided fails", func() {
				context("when build is also set", func() {
					it.Before(func() {
						entryResolver.MergeLayerTypesCall.Returns.Build = true
						pruneProcess.RunCall.Returns.Error = errors.New("prune process failed")
					})

					it("returns an error", func() {
						_, err := build(packit.BuildContext{
							WorkingDir: workingDir,
							Layers:     packit.Layers{Path: layersDir},
							CNBPath:    cnbDir,
							Plan: packit.BuildpackPlan{
								Entries: []packit.BuildpackPlanEntry{
									{Name: "node_modules"},
								},
							},
						})
						Expect(err).To(MatchError("prune process failed"))
					})
				})
			})

			context("when the BOM cannot be generated", func() {
				it.Before(func() {
					sbomGenerator.GenerateCall.Returns.Error = errors.New("failed to generate SBOM")
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						BuildpackInfo: packit.BuildpackInfo{
							SBOMFormats: []string{"application/vnd.cyclonedx+json", "application/spdx+json", "application/vnd.syft+json"},
						},
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{{Name: "node_modules"}},
						},
						Stack: "some-stack",
					})
					Expect(err).To(MatchError("failed to generate SBOM"))
				})
			})

			context("when the BOM cannot be formatted", func() {
				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						WorkingDir: workingDir,
						BuildpackInfo: packit.BuildpackInfo{
							SBOMFormats: []string{"random-format"},
						},
					})
					Expect(err).To(MatchError("unsupported SBOM format: 'random-format'"))
				})
			})

			context("when the node_modules directory cannot be linked", func() {
				it.Before(func() {
					linker.LinkCall.Returns.Error = errors.New("failed to link node_modules")
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError("failed to link node_modules"))
				})
			})

			context("when the linked modules cannot be resolved", func() {
				it.Before(func() {
					symlinkResolver.ResolveCall.Returns.Error = errors.New("failed to resolve module symlinks")
				})

				it("returns an error", func() {
					_, err := build(packit.BuildContext{
						WorkingDir: workingDir,
						Layers:     packit.Layers{Path: layersDir},
						CNBPath:    cnbDir,
						Plan: packit.BuildpackPlan{
							Entries: []packit.BuildpackPlanEntry{
								{Name: "node_modules"},
							},
						},
					})
					Expect(err).To(MatchError("failed to resolve module symlinks"))
				})
			})
		})
	})
}
