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
		err error
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
			app, err = dagger.PackBuild(filepath.Join("testdata", "vendored"), nodeURI, npmURI)
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())

			body, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring("Hello, World!"))
		})

		when("the npm and node buildpacks are cached", func() {
			it("should not reach out to the internet", func() {
				app, err = dagger.NewPack(
					filepath.Join("testdata", "vendored"),
					dagger.RandomImage(),
					dagger.SetBuildpacks(nodeCachedURI, npmCachedURI),
					dagger.SetOffline(),
				).Build()
				Expect(err).ToNot(HaveOccurred())

				Expect(app.Start()).To(Succeed())

				_, _, err = app.HTTPGet("/")
				Expect(app.BuildLogs()).To(ContainSubstring("Reusing cached download from buildpack"))
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	when("when the node_modules are not vendored", func() {
		it("should build a working OCI image for a simple app", func() {
			app, err = dagger.PackBuild(filepath.Join("testdata", "simple_app"), nodeURI, npmURI)
			Expect(err).ToNot(HaveOccurred())

			Expect(app.Start()).To(Succeed())

			body, _, err := app.HTTPGet("/")
			Expect(err).NotTo(HaveOccurred())
			Expect(body).To(ContainSubstring("Hello, World!"))
		})

		when("the npm and node buildpacks are cached", func() {
			it("should install all the node modules", func() {
				app, err = dagger.NewPack(
					filepath.Join("testdata", "simple_app"),
					dagger.RandomImage(),
					dagger.SetBuildpacks(nodeCachedURI, npmCachedURI),
				).Build()
				Expect(err).ToNot(HaveOccurred())

				Expect(app.Start()).To(Succeed())

				_, _, err = app.HTTPGet("/")
				Expect(err).NotTo(HaveOccurred())
				Expect(app.BuildLogs()).To(ContainSubstring("Reusing cached download from buildpack"))
			})
		})
	})

	when("when there are no node modules", func() {
		it("should build a working OCI image for an app without dependencies", func() {
			app, err = dagger.NewPack(
				filepath.Join("testdata", "no_node_modules"),
				dagger.RandomImage(),
				dagger.SetBuildpacks(nodeURI, npmURI),
			).Build()
			Expect(err).ToNot(HaveOccurred())

			app.Start()

			logs, err := app.Logs()
			Expect(err).NotTo(HaveOccurred())
			Expect(logs).To(ContainSubstring("Im a baller"))
		})
	})

	when("when the node modules are partially vendored", func() {
		it("should build a working OCI image for an app that doesn't have a package-lock.json", func() {
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
