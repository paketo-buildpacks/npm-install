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

	when("when the node_modules are vendored", func() {
		it("should build a working OCI image for a simple app", func() {
			app, err := dagger.PackBuild(filepath.Join("testdata", "simple_app_vendored"), nodeBP, bp)
			Expect(err).ToNot(HaveOccurred())
			defer app.Destroy()

			Expect(app.Start()).To(Succeed())

			_, _, err = app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
		})

		//Needs fixing
		//
		//when("the npm and node buildpacks are cached", func() {
		//	it("should not reach out to the internet", func() {
		//		// TODO replace absolute path with wherever we copy nodejs-cnb to
		//		nodeBp, _, err := dagger.PackageCachedBuildpack("/Users/pivotal/workspace/nodejs-cnb")
		//		Expect(err).ToNot(HaveOccurred())
		//
		//		// TODO replace with current root dir somehow
		//		npmBp, _, err := dagger.PackageCachedBuildpack("/Users/pivotal/workspace/npm-cnb")
		//		Expect(err).ToNot(HaveOccurred())
		//
		//		app, err := dagger.PackBuild(filepath.Join("testdata", "simple_app_vendored"), nodeBp, npmBp)
		//		Expect(err).ToNot(HaveOccurred())
		//		defer app.Destroy()
		//
		//		Expect(app.Start()).To(Succeed())
		//
		//		// TODO: add functionality to force network isolation in dagger
		//		_, _, err = app.HTTPGet("/")
		//		Expect(app.BuildLogs()).To(ContainSubstring("Reusing cached download from buildpack"))
		//		Expect(err).NotTo(HaveOccurred())
		//
		//	})
		//})
	})

	when("when the node_modules are not vendored", func() {
		it("should build a working OCI image for a simple app", func() {
			app, err := dagger.PackBuild(filepath.Join("testdata", "simple_app"), nodeBP, bp)
			Expect(err).ToNot(HaveOccurred())
			defer app.Destroy()

			Expect(app.Start()).To(Succeed())

			_, _, err = app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
		})

		// Needs fixing
		//
		//when("the npm and node buildpacks are cached", func() {
		//	it("should install all the node modules", func() {
		//		// TODO replace absolute path with wherever we copy nodejs-cnb to
		//		nodeBp, _, err := dagger.PackageCachedBuildpack("/Users/pivotal/workspace/nodejs-cnb")
		//		Expect(err).ToNot(HaveOccurred())
		//
		//		// TODO replace with current root dir somehow
		//		npmBp, _, err := dagger.PackageCachedBuildpack("/Users/pivotal/workspace/npm-cnb")
		//		Expect(err).ToNot(HaveOccurred())
		//
		//		app, err := dagger.PackBuild(filepath.Join("testdata", "simple_app"), nodeBp, npmBp)
		//		Expect(err).ToNot(HaveOccurred())
		//		defer app.Destroy()
		//
		//		Expect(app.Start()).To(Succeed())
		//
		//		// TODO: add functionality to force network isolation in dagger
		//		_, _, err = app.HTTPGet("/")
		//		Expect(app.BuildLogs()).To(ContainSubstring("Reusing cached download from buildpack"))
		//		Expect(err).NotTo(HaveOccurred())
		//
		//	})
		//})
	})

	when("when there are no node modules", func() {
		it("should build a working OCI image for an app without dependencies", func() {
			_, err := dagger.PackBuild(filepath.Join("testdata", "no_node_modules"), nodeBP, bp)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	when("when the node modules are partially vendored", func() {
		it("should build a working OCI image for an app that doesn't have a package-lock.json", func() {
			app, err := dagger.PackBuild(filepath.Join("testdata", "empty_node_modules"), nodeBP, bp)
			Expect(err).ToNot(HaveOccurred())
			defer app.Destroy()

			Expect(app.Start()).To(Succeed())

			_, _, err = app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
		})
	})
}
