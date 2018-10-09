package detect_test

import (
	"fmt"
	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/npm-cnb/internal/detect"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"os"
	"path/filepath"
)

var _ = Describe("UpdateBuildPlan", func() {
	var (
		err        error
		dir        string
		detectData libbuildpack.Detect
	)

	BeforeEach(func() {
		dir, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())

		detectData = libbuildpack.Detect{
			Application: libbuildpack.Application{Root: dir},
			BuildPlan:   make(map[string]libbuildpack.BuildPlanDependency),
		}
	})

	AfterEach(func() {
		err = os.RemoveAll(dir)
		Expect(err).NotTo(HaveOccurred())
	})

	Context("there is a package.json with a node version in engines", func() {
		const version string = "1.2.3"

		BeforeEach(func() {
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

		It("should create a build plan with the required node version", func() {
			err = detect.UpdateBuildPlan(&detectData)
			Expect(err).NotTo(HaveOccurred())
			Expect(detectData.BuildPlan["node"].Version).To(Equal(version))
		})
	})

	Context("there is no package.json", func() {
		It("returns an error", func() {
			err = detect.UpdateBuildPlan(&detectData)
			Expect(err).To(HaveOccurred())
		})
	})
})
