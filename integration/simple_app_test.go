package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testSimpleApp(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
		app    *dagger.App
	)

	it.After(func() {
		Expect(app.Destroy()).To(Succeed())
	})

	context("when the node_modules are not vendored", func() {
		it("builds a working OCI image for a simple app", func() {
			var err error
			app, err = dagger.NewPack(
				filepath.Join("testdata", "simple_app"),
				dagger.RandomImage(),
				dagger.SetBuildpacks(nodeURI, npmURI),
			).Build()
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())

			body, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring("Hello, World!"))
		})
	})

	context("the app is pushed twice", func() {
		it.Pend("does not reinstall node_modules", func() {
			appDir := filepath.Join("testdata", "simple_app")

			pack := dagger.NewPack(
				appDir,
				dagger.RandomImage(),
				dagger.SetBuildpacks(nodeURI, npmURI),
			)

			var err error
			app, err = pack.Build()
			Expect(err).ToNot(HaveOccurred())

			Expect(app.BuildLogs()).To(MatchRegexp(`Node Modules .*: Contributing to layer`))

			app, err = pack.Build()
			Expect(err).ToNot(HaveOccurred())

			Expect(app.BuildLogs()).To(MatchRegexp(`Node Modules .*: Reusing cached layer`))
			Expect(app.BuildLogs()).NotTo(MatchRegexp(`Node Modules .*: Contributing to layer`))

			Expect(app.Start()).To(Succeed())

			body, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring("Hello, World!"))
		})
	})
}
