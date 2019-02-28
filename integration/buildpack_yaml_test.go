package integration

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

//TODO: DO WE NEED THIS?

func TestBuildpackYAML(t *testing.T) {
	spec.Run(t, "BuildpackYAML", testBuildpackYAML, spec.Report(report.Terminal{}))
}

func testBuildpackYAML(t *testing.T, when spec.G, it spec.S) {
	var (
		bp     string
		nodeBP string
	)

	it.Before(func() {
		RegisterTestingT(t)

		var err error

		err = dagger.BuildCFLinuxFS3()
		Expect(err).ToNot(HaveOccurred())

		bp, err = dagger.PackageBuildpack()
		Expect(err).ToNot(HaveOccurred())

		nodeBP, err = dagger.GetLatestBuildpack("nodejs-cnb")
		Expect(err).ToNot(HaveOccurred())
	})

	it("runs the chosen npm pre and post build scripts", func() {
		app, err := dagger.PackBuild(filepath.Join("testdata", "pre_post_commands"), nodeBP, bp)
		Expect(err).ToNot(HaveOccurred())
		defer app.Destroy()

		Expect(app.Start()).To(Succeed())
		Expect(app.BuildLogs()).To(ContainSubstring("Running my-prebuild (npm)"))
		Expect(app.BuildLogs()).To(ContainSubstring("Running my-postbuild (npm)"))
		Expect(app.BuildLogs()).To(ContainSubstring("postinstall /workspace/app"))

		body, _, err := app.HTTPGet("/")
		Expect(err).NotTo(HaveOccurred())
		Expect(body).To(ContainSubstring("Text: Hello Buildpacks Team"))
		Expect(body).To(ContainSubstring("Text: Goodbye Buildpacks Team"))
	})
}
