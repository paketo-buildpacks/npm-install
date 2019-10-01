package integration_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/npm-cnb/modules"

	"github.com/cloudfoundry/dagger"

	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func init() {
	suite("Integration", testIntegration)
}

func testIntegration(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion
		app    *dagger.App
	)

	it.Before(func() {
		Expect = NewWithT(t).Expect
	})

	it.After(func() {
		if app != nil {
			Expect(app.Destroy()).To(Succeed())
		}
	})

	when("when the node_modules are vendored", func() {
		it("should build a working OCI image for a simple app", func() {
			var err error
			app, err = dagger.PackBuild(filepath.Join("testdata", "vendored"), nodeURI, npmURI)
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())

			body, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring("Hello, World!"))
		})

		// Needs fixing
		when.Pend("the npm and node buildpacks are cached", func() {
			var (
				nodeBp, npmBp string
				err           error
			)

			it.Before(func() {
				nodeBp, _, err = dagger.PackageCachedBuildpack(filepath.Join(bpDir, "..", "node-engine-cnb"))
				Expect(err).ToNot(HaveOccurred())
				defer dagger.DeleteBuildpack(nodeBp)

				npmBp, _, err = dagger.PackageCachedBuildpack(bpDir)
				Expect(err).ToNot(HaveOccurred())
				defer dagger.DeleteBuildpack(npmBp)
			})

			it("should not reach out to the internet", func() {
				var err error
				app, err = dagger.PackBuild(filepath.Join("testdata", "vendored"), nodeBp, npmBp)
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
			var err error
			app, err = dagger.PackBuild(filepath.Join("testdata", "simple_app"), nodeURI, npmURI)
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())

			body, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring("Hello, World!"))
		})

		// Needs fixing
		when.Pend("the npm and node buildpacks are cached", func() {
			var (
				nodeBp, npmBp string
				err           error
			)

			it.Before(func() {
				nodeBp, _, err = dagger.PackageCachedBuildpack(filepath.Join(bpDir, "..", "node-engine-cnb"))
				Expect(err).ToNot(HaveOccurred())
				defer dagger.DeleteBuildpack(nodeBp)

				npmBp, _, err = dagger.PackageCachedBuildpack(bpDir)
				Expect(err).ToNot(HaveOccurred())
				defer dagger.DeleteBuildpack(npmBp)
			})

			it("should install all the node modules", func() {
				var err error
				app, err = dagger.PackBuild(filepath.Join("testdata", "simple_app"), nodeBp, npmBp)
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
			var err error
			app, err = dagger.PackBuild(filepath.Join("testdata", "no_node_modules"), nodeURI, npmURI)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	when("when the node modules are partially vendored", func() {
		it("should build a working OCI image for an app that doesn't have a package-lock.json", func() {
			var err error
			app, err = dagger.PackBuild(filepath.Join("testdata", "empty_node_modules"), nodeURI, npmURI)
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

			var err error
			app, err = dagger.PackBuild(appDir, nodeURI, npmURI)
			Expect(err).ToNot(HaveOccurred())

			Expect(app.BuildLogs()).To(MatchRegexp(fmt.Sprintf("%s .*: Contributing to layer", modules.ModulesMetaName)))

			app, err = dagger.PackBuildNamedImage(app.ImageName, appDir, nodeURI, npmURI)
			Expect(err).ToNot(HaveOccurred())

			Expect(app.BuildLogs()).To(MatchRegexp(fmt.Sprintf("%s .*: Reusing cached layer", modules.ModulesMetaName)))
			Expect(app.BuildLogs()).NotTo(MatchRegexp(fmt.Sprintf("%s .*: Contributing to layer", modules.ModulesMetaName)))

			Expect(app.Start()).To(Succeed())

			body, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring("Hello, World!"))
		})
	})
}
