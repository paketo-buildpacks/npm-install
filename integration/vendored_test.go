package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testVendored(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
		app    *dagger.App
	)

	it.After(func() {
		Expect(app.Destroy()).To(Succeed())
	})

	context("when the node_modules are vendored", func() {
		it("builds a working OCI image for a simple app", func() {
			var err error
			app, err = dagger.NewPack(
				filepath.Join("testdata", "vendored"),
				dagger.RandomImage(),
				dagger.SetBuildpacks(nodeURI, npmURI),
			).Build()
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())

			body, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring("Hello, World!"))
		})

		context("when the npm and node buildpacks are cached", func() {
			it("does not reach out to the internet", func() {
				var err error
				app, err = dagger.NewPack(
					filepath.Join("testdata", "vendored"),
					dagger.RandomImage(),
					dagger.SetBuildpacks(nodeCachedURI, npmCachedURI),
					dagger.SetOffline(),
				).Build()
				Expect(err).ToNot(HaveOccurred())

				Expect(app.Start()).To(Succeed(), func() string { return app.BuildLogs() })

				_, _, err = app.HTTPGet("/")
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})
}
