package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"
	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func init() {
	suite("IncompletePackageJSON", testIncompletePackageJSON)
}

func testIncompletePackageJSON(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect
	})

	when("there is an incomplete package json", func() {
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

		it("builds a working OCI image for a simple app", func() {
			Expect(app.Start()).To(Succeed())

			response, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(ContainSubstring("Hello, World!"))
		})
	})
}
