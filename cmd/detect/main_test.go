package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/cloudfoundry/nodejs-cnb/node"
	"github.com/cloudfoundry/npm-cnb/modules"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDetect(t *testing.T) {
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var factory *test.DetectFactory

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewDetectFactory(t)
	})

	when("there is a package.json with a node version in engines", func() {
		const version string = "1.2.3"

		it("should pass", func() {
			packageJSONString := fmt.Sprintf(`{"engines": {"node" : "%s"}}`, version)
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "package.json"), packageJSONString)

			buildplan := getStandardBuildplanWithNodeVersion(version)
			runDetectAndExpectBuildplan(factory, buildplan)
		})
	})

	when("there is a package.json", func() {
		it("should pass with the default version of node", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "package.json"), "{}")

			buildplan := getStandardBuildplanWithNodeVersion("")
			runDetectAndExpectBuildplan(factory, buildplan)

		})
	})

	when("there is no package.json", func() {
		it("should fail", func() {
			code, err := runDetect(factory.Detect)
			Expect(err).To(HaveOccurred())
			Expect(code).To(Equal(detect.FailStatusCode))
		})
	})

	when("When .nvmrc is present", func() {
		when("nvmrc is empty", func() {
			it("should fail", func() {
				test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, ".nvmrc"), "")
				code, err := runDetect(factory.Detect)
				Expect(err).To(HaveOccurred())
				Expect(code).To(Equal(detect.FailStatusCode))
			})
		})

		when("nvmrc is present and engines field in package.json is present", func() {
			it("selects the version from the engines field in packages.json", func() {
				packageJSONVersion := "10.0.0"
				packageJSONString := fmt.Sprintf(`{"engines": {"node" : "%s"}}`, packageJSONVersion)
				test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "package.json"), packageJSONString)

				nvmrcString := "10.2.3"
				test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, ".nvmrc"), nvmrcString)

				buildplan := getStandardBuildplanWithNodeVersion(packageJSONVersion)
				runDetectAndExpectBuildplan(factory, buildplan)
			})
		})

		when("nvmrc is present and engines field in package.json is missing", func() {
			it("selects the version in nvmrc", func() {
				packageJSONString := fmt.Sprintf(`{"engines": {"node" : "%s"}}`, "")
				test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "package.json"), packageJSONString)

				nvmrcString := "10.2.3"
				test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, ".nvmrc"), nvmrcString)

				buildplan := getStandardBuildplanWithNodeVersion(nvmrcString)
				runDetectAndExpectBuildplan(factory, buildplan)
			})
		})
	})

	when("there are .nvmrc contents", func() {
		when("the .nvmrc contains only digits", func() {
			it("will trim and transform nvmrc to appropriate semver for Masterminds semver library", func() {
				testCases := [][]string{
					{"10", "10.*.*"},
					{"10.2", "10.2.*"},
					{"v10", "10.*.*"},
					{"10.2.3", "10.2.3"},
					{"v10.2.3", "10.2.3"},
					{"10.1.1", "10.1.1"},
				}

				for _, testCase := range testCases {
					test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, ".nvmrc"), testCase[0])
					Expect(LoadNvmrc(factory.Detect)).To(Equal(testCase[1]), fmt.Sprintf("failed for test case %s : %s", testCase[0], testCase[1]))
				}
			})
		})

		when("the .nvmrc contains lts/something", func() {
			it("will read and trim lts versions to appropriate semver for Masterminds semver library", func() {
				testCases := [][]string{
					{"lts/*", "10.*.*"},
					{"lts/argon", "4.*.*"},
					{"lts/boron", "6.*.*"},
					{"lts/carbon", "8.*.*"},
					{"lts/dubnium", "10.*.*"},
				}

				for _, testCase := range testCases {
					test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, ".nvmrc"), testCase[0])
					Expect(LoadNvmrc(factory.Detect)).To(Equal(testCase[1]), fmt.Sprintf("failed for test case %s : %s", testCase[0], testCase[1]))
				}
			})
		})

		when("the .nvmrc contains 'node'", func() {
			it("should read and trim lts versions", func() {
				test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, ".nvmrc"), "node")
				Expect(LoadNvmrc(factory.Detect)).To(Equal("*"))
			})
		})

		when("given an invalid .nvmrc", func() {
			it("validate should be fail", func() {
				invalidVersions := []string{"11.4.x", "invalid", "~1.1.2", ">11.0", "< 11.4.2", "^1.2.3", "11.*.*", "10.1.X", "lts/invalidname"}
				InvalidVersionError := errors.New("invalid version Invalid Semantic Version specified in .nvmrc")
				for _, version := range invalidVersions {
					test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, ".nvmrc"), version)
					parsedVersion, err := LoadNvmrc(factory.Detect)
					Expect(err).To(Equal(InvalidVersionError))
					Expect(parsedVersion).To(BeEmpty())
				}
			})
		})
	})
}

func runDetectAndExpectBuildplan(factory *test.DetectFactory, buildplan buildplan.BuildPlan) {
	code, err := runDetect(factory.Detect)
	Expect(err).NotTo(HaveOccurred())

	Expect(code).To(Equal(detect.PassStatusCode))

	Expect(factory.Output).To(Equal(buildplan))
}

func getStandardBuildplanWithNodeVersion(version string) buildplan.BuildPlan {
	return buildplan.BuildPlan{
		node.Dependency: buildplan.Dependency{
			Version:  version,
			Metadata: buildplan.Metadata{"build": true, "launch": true},
		},
		modules.Dependency: buildplan.Dependency{
			Metadata: buildplan.Metadata{"launch": true},
		},
	}
}
