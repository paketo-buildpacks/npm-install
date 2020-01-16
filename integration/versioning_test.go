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

	context("when using a nvmrc file", func() {
		const nvmrcVersion = `12.\d+\.\d+`

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
