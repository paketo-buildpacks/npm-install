package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/cloudfoundry/npm-cnb/modules"
	"github.com/cloudfoundry/npm-cnb/node"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDetect(t *testing.T) {
	RegisterTestingT(t)
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var (
		err     error
		dir     string
		factory *test.DetectFactory
	)

	it.Before(func() {
		dir, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())

		factory = test.NewDetectFactory(t)
		factory.Detect.Application.Root = dir
	})

	it.After(func() {
		Expect(os.RemoveAll(dir)).To(Succeed())
	})

	when("there is a package.json with a node version in engines", func() {
		const version string = "1.2.3"

		it.Before(func() {
			packageJSONString := fmt.Sprintf(`{"engines": {"node" : "%s"}}`, version)
			Expect(ioutil.WriteFile(filepath.Join(dir, "package.json"), []byte(packageJSONString), 0666)).To(Succeed())
		})

		it("should pass", func() {
			code, err := runDetect(factory.Detect)
			Expect(err).NotTo(HaveOccurred())

			Expect(code).To(Equal(detect.PassStatusCode))

			test.BeBuildPlanLike(t, factory.Output, buildplan.BuildPlan{
				node.Dependency: buildplan.Dependency{
					Version:  version,
					Metadata: buildplan.Metadata{"build": true, "launch": true},
				},
				modules.Dependency: buildplan.Dependency{
					Metadata: buildplan.Metadata{"launch": true},
				},
			})

		})
	})

	when("there is a package.json", func() {
		it.Before(func() {
			Expect(ioutil.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0666)).To(Succeed())
		})

		it("should pass with the default version of node", func() {
			code, err := runDetect(factory.Detect)
			Expect(err).NotTo(HaveOccurred())

			Expect(code).To(Equal(detect.PassStatusCode))

			test.BeBuildPlanLike(t, factory.Output, buildplan.BuildPlan{
				node.Dependency: buildplan.Dependency{
					Version:  "",
					Metadata: buildplan.Metadata{"build": true, "launch": true},
				},
				modules.Dependency: buildplan.Dependency{
					Metadata: buildplan.Metadata{"launch": true},
				},
			})

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
