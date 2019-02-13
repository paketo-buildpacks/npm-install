package main

import (
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
		when("nvmrc is present and engines field in package.json is present", func() {
			it("selects the version from the engines field in packages.json", func() {
				packageJSONVersion := "10.0.0"
				packageJSONString := fmt.Sprintf(`{"engines": {"node" : "%s"}}`, packageJSONVersion)
				test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "package.json"), packageJSONString)

				nvmrcVersion := "10.2.3"
				factory.AddBuildPlan(node.Dependency, buildplan.Dependency{
					Version:  nvmrcVersion,
					Metadata: buildplan.Metadata{"launch": true},
				})

				buildplan := getStandardBuildplanWithNodeVersion(packageJSONVersion)
				runDetectAndExpectBuildplan(factory, buildplan)
			})
		})

		when("nvmrc is present and engines field in package.json is missing", func() {
			it("selects the version in nvmrc", func() {
				packageJSONString := fmt.Sprintf(`{"engines": {"node" : "%s"}}`, "")
				test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "package.json"), packageJSONString)

				nvmrcVersion := "10.2.3"
				factory.AddBuildPlan(node.Dependency, buildplan.Dependency{
					Version:  nvmrcVersion,
					Metadata: buildplan.Metadata{"launch": true},
				})

				buildplan := getStandardBuildplanWithNodeVersion(nvmrcVersion)
				runDetectAndExpectBuildplan(factory, buildplan)
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
