package integration

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestIncompletePackageJSONIntegration(t *testing.T) {
	spec.Run(t, "IncompletePackageJSONIntegration", incompletePackageJSONIntegration, spec.Report(report.Terminal{}))
}

func incompletePackageJSONIntegration(t *testing.T, when spec.G, it spec.S) {
	var (
		bp     string
		nodeBP string
	)

	it.Before(func() {
		RegisterTestingT(t)

		var err error

		bp, err = dagger.PackageBuildpack()
		Expect(err).ToNot(HaveOccurred())

		nodeBP, err = dagger.GetLatestBuildpack("nodejs-cnb")
		Expect(err).ToNot(HaveOccurred())
	})

	when("when there is an empty node_modules", func() {
		it("should build a working OCI image for a simple app", func() {
			app, err := dagger.PackBuild(filepath.Join("testdata", "incomplete_package_json"), nodeBP, bp)
			Expect(err).ToNot(HaveOccurred())
			defer app.Destroy()

			Expect(app.Start()).To(Succeed())
			response, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(ContainSubstring("Hello, World!"))
		})
	})
}
