package integration

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestIntegration(t *testing.T) {
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	it.Before(func() {
		RegisterTestingT(t)
	})

	when("when the node_modules are vendored", func() {
		it("should build a working OCI image for a simple app", func() {
			bp, err := dagger.PackageBuildpack()
			Expect(err).ToNot(HaveOccurred())

			nodeBP, err := dagger.GetRemoteBuildpack("https://github.com/cloudfoundry/nodejs-cnb/releases/download/v0.0.2/nodejs-cnb.tgz")
			Expect(err).ToNot(HaveOccurred())

			app, err := dagger.PackBuild(filepath.Join("fixtures", "simple_app_vendored"), nodeBP, bp)
			Expect(err).ToNot(HaveOccurred())
			defer app.Destroy()

			Expect(app.Start()).To(Succeed())

			Expect(app.HTTPGet("/")).To(Succeed())
		})
	})

	when("when the node_modules are not vendored", func() {
		it("should build a working OCI image for a simple app", func() {
			bp, err := dagger.PackageBuildpack()
			Expect(err).ToNot(HaveOccurred())

			nodeBP, err := dagger.GetRemoteBuildpack("https://github.com/cloudfoundry/nodejs-cnb/releases/download/v0.0.2/nodejs-cnb.tgz")
			Expect(err).ToNot(HaveOccurred())

			app, err := dagger.PackBuild(filepath.Join("fixtures", "simple_app"), nodeBP, bp)
			Expect(err).ToNot(HaveOccurred())
			defer app.Destroy()

			Expect(app.Start()).To(Succeed())

			Expect(app.HTTPGet("/")).To(Succeed())
		})
	})
}
