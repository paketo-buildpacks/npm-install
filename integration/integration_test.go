package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	bpDir, npmURI, nodeURI string
)

var suite = spec.New("Integration", spec.Report(report.Terminal{}))

func init() {
	suite("Integration", testIntegration)
}

func TestIntegration(t *testing.T) {
	var err error
	Expect := NewWithT(t).Expect
	bpDir, err = dagger.FindBPRoot()
	Expect(err).NotTo(HaveOccurred())
	npmURI, err = dagger.PackageBuildpack(bpDir)
	Expect(err).ToNot(HaveOccurred())
	defer dagger.DeleteBuildpack(npmURI)

	nodeURI, err = dagger.GetLatestBuildpack("node-engine-cnb")
	Expect(err).ToNot(HaveOccurred())
	defer dagger.DeleteBuildpack(nodeURI)

	suite.Run(t)
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion
		app *dagger.App
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect
	})

	it.After(func() {
		if app != nil {
			app.Destroy()
		}
	})

	when("when the node_modules are vendored", func() {
		it("should build a working OCI image for a simple app", func() {
			app, err := dagger.PackBuild(filepath.Join("testdata", "vendored"), nodeURI, npmURI)
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())

			body, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring("Hello, World!"))
		})

		// Needs fixing
		when.Pend("the npm and node buildpacks are cached", func() {
			var(
				nodeBp, npmBp string
				err error
			)
			it.Before(func() {
				nodeBp, _, err = dagger.PackageCachedBuildpack(filepath.Join(bpDir,"..","node-engine-cnb"))
				Expect(err).ToNot(HaveOccurred())
				defer dagger.DeleteBuildpack(nodeBp)

				npmBp, _, err = dagger.PackageCachedBuildpack(bpDir)
				Expect(err).ToNot(HaveOccurred())
				defer dagger.DeleteBuildpack(npmBp)
			})

			it("should not reach out to the internet", func() {
				app, err := dagger.PackBuild(filepath.Join("testdata", "vendored"), nodeBp, npmBp)
				Expect(err).ToNot(HaveOccurred())

				Expect(app.Start()).To(Succeed())

				// TODO: add functionality to force network isolation in dagger
				_, _, err = app.HTTPGet("/")
				Expect(app.BuildLogs()).To(ContainSubstring("Reusing cached download from buildpack"))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	when("when the node_modules are not vendored", func() {
		it("should build a working OCI image for a simple app", func() {
			app, err := dagger.PackBuild(filepath.Join("testdata", "simple_app"), nodeURI, npmURI)
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())

			body, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring("Hello, World!"))

		})

		// Needs fixing
		when.Pend("the npm and node buildpacks are cached", func() {
			var(
				nodeBp, npmBp string
				err error
			)
			it.Before(func() {
				nodeBp, _, err = dagger.PackageCachedBuildpack(filepath.Join(bpDir,"..","node-engine-cnb"))
				Expect(err).ToNot(HaveOccurred())
				defer dagger.DeleteBuildpack(nodeBp)

				npmBp, _, err = dagger.PackageCachedBuildpack(bpDir)
				Expect(err).ToNot(HaveOccurred())
				defer dagger.DeleteBuildpack(npmBp)
			})

			it("should install all the node modules", func() {
				app, err := dagger.PackBuild(filepath.Join("testdata", "simple_app"), nodeBp, npmBp)
				Expect(err).ToNot(HaveOccurred())

				Expect(app.Start()).To(Succeed())

				_, _, err = app.HTTPGet("/")
				Expect(app.BuildLogs()).To(ContainSubstring("Reusing cached download from buildpack"))
				Expect(err).NotTo(HaveOccurred())
			//	TODO expect logs to contain something about downloading modules
			})
		})
	})

	when("when there are no node modules", func() {
		it("should build a working OCI image for an app without dependencies", func() {
			_, err := dagger.PackBuild(filepath.Join("testdata", "no_node_modules"), nodeURI, npmURI)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	when("when the node modules are partially vendored", func() {
		it("should build a working OCI image for an app that doesn't have a package-lock.json", func() {
			app, err := dagger.PackBuild(filepath.Join("testdata", "empty_node_modules"), nodeURI, npmURI)
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())

			body, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring("Hello, World!"))
		})
	})

	when("the app is pushed twice", func() {
		it("does not reinstall node_modules", func() {
			appDir := filepath.Join("testdata", "simple_app")
			app, err := dagger.PackBuild(appDir, nodeURI, npmURI)
			Expect(err).ToNot(HaveOccurred())

			Expect(app.BuildLogs()).To(MatchRegexp("node_modules .*: Contributing to layer"))

			app, err = dagger.PackBuildNamedImage(app.ImageName, appDir, nodeURI, npmURI)

			Expect(app.BuildLogs()).To(MatchRegexp("node_modules .*: Reusing cached layer"))
			Expect(app.BuildLogs()).NotTo(MatchRegexp("node_modules .*: Contributing to layer"))

			Expect(app.Start()).To(Succeed())

			body, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring("Hello, World!"))
		})
	})
}
