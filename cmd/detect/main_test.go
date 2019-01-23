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

		it.Before(func() {
			packageJSONString := fmt.Sprintf(`{"engines": {"node" : "%s"}}`, version)
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "package.json"), packageJSONString)
		})

		it("should pass", func() {
			code, err := runDetect(factory.Detect)
			Expect(err).NotTo(HaveOccurred())

			Expect(code).To(Equal(detect.PassStatusCode))

			Expect(factory.Output).To(Equal(buildplan.BuildPlan{
				node.Dependency: buildplan.Dependency{
					Version:  version,
					Metadata: buildplan.Metadata{"build": true, "launch": true},
				},
				modules.Dependency: buildplan.Dependency{
					Metadata: buildplan.Metadata{"launch": true},
				},
			}))

		})
	})

	when("there is a package.json", func() {
		it.Before(func() {
			test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "package.json"), "{}")
		})

		it("should pass with the default version of node", func() {
			code, err := runDetect(factory.Detect)
			Expect(err).NotTo(HaveOccurred())

			Expect(code).To(Equal(detect.PassStatusCode))

			Expect(factory.Output).To(Equal(buildplan.BuildPlan{
				node.Dependency: buildplan.Dependency{
					Version:  "",
					Metadata: buildplan.Metadata{"build": true, "launch": true},
				},
				modules.Dependency: buildplan.Dependency{
					Metadata: buildplan.Metadata{"launch": true},
				},
			}))

		})
	})

	when("there is no package.json", func() {
		it("should fail", func() {
			code, err := runDetect(factory.Detect)
			Expect(err).To(HaveOccurred())
			Expect(code).To(Equal(detect.FailStatusCode))
		})
	})
}
