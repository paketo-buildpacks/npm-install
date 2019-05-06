package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"

	. "github.com/onsi/gomega"
	"github.com/sclevine/spec"
)

func testBuildpackYAML(t *testing.T, when spec.G, it spec.S) {
	var Expect func(interface{}, ...interface{}) Assertion

	it.Before(func() {
		Expect = NewWithT(t).Expect
	})

	it("runs the chosen npm pre and post build scripts", func() {
		app, err := dagger.PackBuild(filepath.Join("testdata", "pre_post_commands"), nodejsCNB, bp)
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
