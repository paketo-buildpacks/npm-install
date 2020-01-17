package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPrePostScriptRebuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
		app    *dagger.App
	)

	it.After(func() {
		Expect(app.Destroy()).To(Succeed())
	})

	context("when the npm and node buildpacks are cached", func() {
		it("does not reach out to the internet", func() {
			var err error
			app, err = dagger.NewPack(
				filepath.Join("testdata", "pre_post_scripts_vendored"),
				dagger.RandomImage(),
				dagger.SetBuildpacks(nodeCachedURI, npmCachedURI),
				dagger.SetOffline(),
				dagger.SetVerbose(),
			).Build()
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed(), func() string { return app.BuildLogs() })

			body, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring("running preinstall script"))
			Expect(body).To(ContainSubstring("running postinstall script"))
		})
	})
}
