package detect_test

import (
	"fmt"
	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/npm-cnb/detect"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestUnitDetect(t *testing.T) {
	RegisterTestingT(t)
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var (
		err        error
		dir        string
		detectData libbuildpack.Detect
	)

	it.Before(func() {
		dir, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())

		detectData = libbuildpack.Detect{
			Application: libbuildpack.Application{Root: dir},
			BuildPlan:   make(map[string]libbuildpack.BuildPlanDependency),
		}
	})

	it.After(func() {
		err = os.RemoveAll(dir)
		Expect(err).NotTo(HaveOccurred())
	})

	when("there is a package.json with a node version in engines", func() {
		const version string = "1.2.3"

		it.Before(func() {
			packageJSONString := fmt.Sprintf(`{
				"name": "bson-test",
				"version": "1.0.0",
				"description": "",
				"main": "index.js",
				"scripts": {
				"start": "node index.js"
			},
				"author": "",
				"license": "ISC",
				"dependencies": {
				"bson-ext": "^0.1.13"
			},
				"engines": {
				"node" : "%s"
			}
			}`, version)
			err = ioutil.WriteFile(
				filepath.Join(dir, "package.json"),
				[]byte(packageJSONString),
				0666,
			)
			Expect(err).NotTo(HaveOccurred())
		})

		it("should create a build plan with the required node version", func() {
			err = detect.UpdateBuildPlan(&detectData)
			Expect(err).NotTo(HaveOccurred())
			Expect(detectData.BuildPlan["node"].Version).To(Equal(version))
		})
	})

	when("there is no package.json", func() {
		it("returns an error", func() {
			err = detect.UpdateBuildPlan(&detectData)
			Expect(err).To(HaveOccurred())
		})
	})
}
