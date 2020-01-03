package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testIncompletePackageJSON(t *testing.T, context spec.G, it spec.S) {
	var Expect = NewWithT(t).Expect

	context("when there is an incomplete package json", func() {
		var app *dagger.App

		it.Before(func() {
			var err error
			app, err = dagger.NewPack(
				filepath.Join("testdata", "incomplete_package_json"),
				dagger.RandomImage(),
				dagger.SetBuildpacks(nodeURI, npmURI),
			).Build()
			Expect(err).ToNot(HaveOccurred())
		})

		it.After(func() {
			Expect(app.Destroy()).To(Succeed())
		})

		it.Pend("builds a working OCI image for a simple app", func() {
			Expect(app.Start()).To(Succeed())

			response, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(ContainSubstring("Hello, World!"))
		})
	})
}
