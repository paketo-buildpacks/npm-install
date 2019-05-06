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
		app *dagger.App
		err error
		Expect func(interface{}, ...interface{}) Assertion
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect
	})

	it.After(func(){
		if app != nil {
			app.Destroy()
		}
	})

	when("npm version minor patch is floated", func() {
		it("should build a working OCI image, but not respect specified npm version", func() {
			app, err = dagger.PackBuild(filepath.Join("testdata", "npm_version_with_minor_x"), nodeURI, npmURI)
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())
			resp, err := app.HTTPGetBody("/")
			matchable := `Hello, World! From npm version: \d+\.\d+.\d+`
			unmatchable := `Hello, World! From npm version: 99.99.99`
			Expect(resp).To(MatchRegexp(matchable))
			Expect(resp).NotTo(MatchRegexp(unmatchable))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	when("using a nvmrc file", func() {
		const nvmrcVersion = `8.\d+\.\d+`

		it("package.json takes precedence over it", func() {
			app, err = dagger.PackBuild(filepath.Join("testdata", "simple_app_with_nvmrc"), nodeURI, npmURI)
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())
			resp, err := app.HTTPGetBody("/")
			matchable := `Hello, World! From node version: v\d+\.\d+.\d+`
			unmatchable := `Hello, World! From node version: v` + nvmrcVersion
			Expect(resp).To(MatchRegexp(matchable))
			Expect(resp).NotTo(MatchRegexp(unmatchable))
			Expect(err).NotTo(HaveOccurred())
		})

		it("it is honored if there package.json doesn't have an engine", func() {
			app, err = dagger.PackBuild(filepath.Join("testdata", "simple_app_with_nvmrc_and_no_engine"), nodeURI, npmURI)
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())
			resp, err := app.HTTPGetBody("/")
			Expect(strings.TrimSpace(resp)).To(MatchRegexp(`Hello, World! From node version: v` + nvmrcVersion))
			Expect(err).NotTo(HaveOccurred())
		})
	})
}
