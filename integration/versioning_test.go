package integration

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/cloudfoundry/dagger"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestVersioningIntegration(t *testing.T) {
	spec.Run(t, "Integration", testVersioningIntegration, spec.Report(report.Terminal{}))
}

func testVersioningIntegration(t *testing.T, when spec.G, it spec.S) {
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

	when("when using node version 6", func() {
		it("should build a working OCI image for a simple Node 6 app", func() {
			app, err := dagger.PackBuild(filepath.Join("testdata", "node_version_6"), nodeBP, bp)
			Expect(err).ToNot(HaveOccurred())
			defer app.Destroy()

			Expect(app.Start()).To(Succeed())
			Expect(app.BuildStdout.String()).To(MatchRegexp(`NodeJS[^\.\n]*6\.`)) //Allows it to ignore the control characters for color
			Expect(app.HTTPGetBody("/")).To(ContainSubstring("Hello, World!"))
		})
	})

	when("npm version minor patch is floated", func() {
		it("should build a working OCI image, but not respect specified npm version", func() {
			app, err := dagger.PackBuild(filepath.Join("testdata", "npm_version_with_minor_x"), nodeBP, bp)
			Expect(err).ToNot(HaveOccurred())
			defer app.Destroy()

			Expect(app.Start()).To(Succeed())
			resp, err := app.HTTPGetBody("/")
			matchable := `Hello, World! From npm version: \d+\.\d+.\d+`
			unmatchable := `Hello, World! From npm version: 99.99.99`
			Expect(resp).To(MatchRegexp(matchable))
			Expect(resp).NotTo(MatchRegexp(unmatchable))
			Expect(err).NotTo(HaveOccurred())
		})
	})

	when("using a nvmrc file", func() {
		const nvmrcVersion = "8.15.0"

		it("package.json takes precedence over it", func() {
			app, err := dagger.PackBuild(filepath.Join("testdata", "simple_app_with_nvmrc"), nodeBP, bp)
			Expect(err).ToNot(HaveOccurred())
			defer app.Destroy()

			Expect(app.Start()).To(Succeed())
			resp, err := app.HTTPGetBody("/")
			matchable := `Hello, World! From node version: v\d+\.\d+.\d+`
			unmatchable := `Hello, World! From node version: v` + nvmrcVersion
			Expect(resp).To(MatchRegexp(matchable))
			Expect(resp).NotTo(MatchRegexp(unmatchable))
			Expect(err).NotTo(HaveOccurred())
		})

		it("it is honored if there package.json doesn't have an engine", func() {
			app, err := dagger.PackBuild(filepath.Join("testdata", "simple_app_with_nvmrc_and_no_engine"), nodeBP, bp)
			Expect(err).ToNot(HaveOccurred())
			defer app.Destroy()

			Expect(app.Start()).To(Succeed())
			resp, err := app.HTTPGetBody("/")
			Expect(strings.TrimSpace(resp)).To(Equal(`Hello, World! From node version: v` + nvmrcVersion))
			Expect(err).NotTo(HaveOccurred())
		})
	})
}
