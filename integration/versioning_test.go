package integration_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudfoundry/dagger"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testVersioning(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
		app    *dagger.App
	)

	it.After(func() {
		Expect(app.Destroy()).To(Succeed())
	})

	context("when the npm version minor patch is floated", func() {
		// NOTE: this regression test was based on v2 buildpack functionality that
		// installed a version of npm that was specified by the user in the
		// package.json file. We seem to have skipped this functionality in the
		// CNB. We are waiting on feedback for whether we should be implementing
		// that behavior here and adding test coverage for it, or removing this as
		// an unnecessary test.
		it.Pend("builds a working OCI image, but not respect specified npm version", func() {
			var err error
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

	context("when using a nvmrc file", func() {
		const nvmrcVersion = `8.\d+\.\d+`

		it("package.json takes precedence over it", func() {
			var err error
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
			var err error
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
