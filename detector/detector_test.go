package detector_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/npm-cnb/detector"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/cloudfoundry/npm-cnb/modules"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDetect(t *testing.T) {
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var (
		factory *test.DetectFactory
		d       detector.Detector
	)

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewDetectFactory(t)
		d = detector.Detector{}
	})

	when("there is a package.json with a node version in engines", func() {
		const version string = "1.2.3"

		it("should pass", func() {
			packageJSONString := fmt.Sprintf(`{"engines": {"node" : "%s"}}`, version)
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "package.json"), packageJSONString)

			plan := getStandardBuildplanWithNodeVersion(version)
			runDetectAndExpectBuildplan(factory, d, plan)
		})
	})

	when("there is a package.json with no node version", func() {
		it("should pass with empty version of node", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "package.json"), "{}")

			plan := getStandardBuildplanWithNodeVersion("")
			runDetectAndExpectBuildplan(factory, d, plan)
		})
	})

	when("there is no package.json", func() {
		it("should fail", func() {
			code, err := d.RunDetect(factory.Detect)
			Expect(err).NotTo(HaveOccurred())
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
				factory.AddBuildPlan(modules.NodeDependency, buildplan.Dependency{
					Version:  nvmrcVersion,
					Metadata: buildplan.Metadata{"launch": true},
				})

				plan := getStandardBuildplanWithNodeVersion(packageJSONVersion)
				runDetectAndExpectBuildplan(factory, d, plan)
			})
		})

		when("nvmrc is present and engines field in package.json is missing", func() {
			it("selects the version in nvmrc", func() {
				packageJSONString := `{"engines": {"node" : ""}}`
				test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "package.json"), packageJSONString)

				nvmrcVersion := "10.2.3"
				factory.AddBuildPlan(modules.NodeDependency, buildplan.Dependency{
					Version:  nvmrcVersion,
					Metadata: buildplan.Metadata{"launch": true},
				})

				plan := getStandardBuildplanWithNodeVersion(nvmrcVersion)
				runDetectAndExpectBuildplan(factory, d, plan)
			})
		})
	})
}

func runDetectAndExpectBuildplan(factory *test.DetectFactory, d detector.Detector, buildplan buildplan.BuildPlan) {
	code, err := d.RunDetect(factory.Detect)
	Expect(err).NotTo(HaveOccurred())

	Expect(code).To(Equal(detect.PassStatusCode))

	Expect(factory.Output).To(Equal(buildplan))
}

func getStandardBuildplanWithNodeVersion(version string) buildplan.BuildPlan {
	return buildplan.BuildPlan{
		modules.NodeDependency: buildplan.Dependency{
			Version:  version,
			Metadata: buildplan.Metadata{"build": true, "launch": true},
		},
		modules.Dependency: buildplan.Dependency{
			Metadata: buildplan.Metadata{"launch": true},
		},
	}
}
