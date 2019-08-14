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

			plan := getStandardBuildplanWithNodeVersion(version, "package.json")
			runDetectAndExpectBuildplan(factory, d, plan)
		})
	})

	when("there is a package.json with no node version", func() {
		it("should pass with empty version of node", func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "package.json"), "{}")

			plan := getStandardBuildplanWithNodeVersion("", "")
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

	when("package.json exists", func() {
		when("node version is specified in the engines field", func() {
			var packageJSONPath, packageJSONVersion string

			it.Before(func() {
				packageJSONVersion = "10.0.0"
				packageJSONString := fmt.Sprintf(`{"engines": {"node" : "%s"}}`, packageJSONVersion)
				packageJSONPath = filepath.Join(factory.Detect.Application.Root, "package.json")
				test.WriteFile(t, packageJSONPath, packageJSONString)
			})

			it("selects the node version from the engines field in package.json", func() {
				plan := getStandardBuildplanWithNodeVersion(packageJSONVersion, "package.json")
				runDetectAndExpectBuildplan(factory, d, plan)
			})
		})

		when("node version is not specified in the engines field", func() {
			var packageJSONPath string

			it.Before(func() {
				packageJSONString := `{"engines": {"node" : ""}}`
				packageJSONPath = filepath.Join(factory.Detect.Application.Root, "package.json")
				test.WriteFile(t, packageJSONPath, packageJSONString)
			})

			it("does not request a specific version of node", func() {
				plan := getStandardBuildplanWithNodeVersion("", "")
				runDetectAndExpectBuildplan(factory, d, plan)
			})
		})
	})
}

func runDetectAndExpectBuildplan(factory *test.DetectFactory, d detector.Detector, buildplan buildplan.Plan) {
	code, err := d.RunDetect(factory.Detect)
	Expect(err).NotTo(HaveOccurred())

	Expect(code).To(Equal(detect.PassStatusCode))

	Expect(factory.Plans.Plan).To(Equal(buildplan))
}

func getStandardBuildplanWithNodeVersion(version, versionSource string) buildplan.Plan {
	return buildplan.Plan{
		Provides: []buildplan.Provided{{Name: modules.Dependency}},
		Requires: []buildplan.Required{
			{
				Name:     modules.NodeDependency,
				Version:  version,
				Metadata: buildplan.Metadata{
					"build": true,
					"launch": true,
					"version-source": versionSource,
				},
			},
			{
				Name:     modules.Dependency,
				Metadata: buildplan.Metadata{"launch": true},
			},
		},
	}
}
