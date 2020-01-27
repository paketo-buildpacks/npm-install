package integration_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudfoundry/dagger"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func init() {
	suite("Versioning", testVersioning)
}

func testVersioning(t *testing.T, when spec.G, it spec.S) {
	var (
		app    *dagger.App
		Expect func(interface{}, ...interface{}) Assertion
		err    error
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect
	})

	it.After(func() {
		if app != nil {
			Expect(app.Destroy()).To(Succeed())
		}
	})

	when("npm version minor patch is floated", func() {
		it("should build a working OCI image, but not respect specified npm version", func() {
			app, err = dagger.NewPack(
				filepath.Join("testdata", "npm_version_with_minor_x"),
				dagger.RandomImage(),
				dagger.SetBuildpacks(nodeURI, npmURI),
			).Build()
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())

			resp, err := app.HTTPGetBody("/")
			Expect(err).NotTo(HaveOccurred())

			Expect(resp).To(MatchRegexp(`Hello, World! From npm version: \d+\.\d+.\d+`))
			Expect(resp).NotTo(MatchRegexp(`Hello, World! From npm version: 99.99.99`))
		})
	})

	when("using a nvmrc file", func() {
		const nvmrcVersion = `12.\d+\.\d+`

		it("package.json takes precedence over it", func() {
			app, err = dagger.NewPack(
				filepath.Join("testdata", "with_nvmrc"),
				dagger.RandomImage(),
				dagger.SetBuildpacks(nodeURI, npmURI),
			).Build()
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())

			resp, err := app.HTTPGetBody("/")
			Expect(err).NotTo(HaveOccurred())

			Expect(resp).To(MatchRegexp(`Hello, World! From node version: v\d+\.\d+.\d+`))
			Expect(resp).NotTo(MatchRegexp(`Hello, World! From node version: v` + nvmrcVersion))
		})

		it("is honored if the package.json doesn't have an engine version", func() {
			app, err = dagger.NewPack(
				filepath.Join("testdata", "with_nvmrc_and_no_engine"),
				dagger.RandomImage(),
				dagger.SetBuildpacks(nodeURI, npmURI),
			).Build()
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())

			resp, err := app.HTTPGetBody("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(strings.TrimSpace(resp)).To(MatchRegexp(`Hello, World! From node version: v` + nvmrcVersion))
		})
	})
}
