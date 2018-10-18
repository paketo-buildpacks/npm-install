package acceptance

import (
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"path/filepath"
	"testing"

	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/dagger"
	. "github.com/onsi/gomega"
)

func TestAcceptance(t *testing.T) {
	RegisterTestingT(t)
	spec.Run(t, "Acceptance", testAcceptance, spec.Report(report.Terminal{}))
}

func testAcceptance(t *testing.T, when spec.G, it spec.S) {
	var (
		rootDir    string
		dagg       *dagger.Dagger
		buildpacks []struct {
			ID  string
			URI string
		}
		groups          []dagger.Group
		builderMetadata dagger.BuilderMetadata
	)

	it.Before(func() {
		var err error

		rootDir, err = dagger.FindRoot()
		Expect(err).ToNot(HaveOccurred())

		dagg, err = dagger.NewDagger(rootDir)
		Expect(err).ToNot(HaveOccurred())

		buildpacks = []struct {
			ID  string
			URI string
		}{
			{
				ID:  "org.cloudfoundry.buildpacks.nodejs",
				URI: "https://github.com/cloudfoundry/nodejs-cnb/releases/download/v0.0.1-alpha/nodejs-cnb.tgz",
			},
			{
				ID:  "org.cloudfoundry.buildpacks.npm",
				URI: "file://" + dagg.BuildpackDir,
			},
		}

		groups = []dagger.Group{
			{
				Buildpacks: []libbuildpack.BuildpackInfo{
					{
						ID:      "org.cloudfoundry.buildpacks.nodejs",
						Name:    "nodejs",
						Version: "0.0.1",
					},
					{
						ID:      "org.cloudfoundry.buildpacks.npm",
						Name:    "npm",
						Version: "0.0.1",
					},
				},
			},
		}

		builderMetadata = dagger.BuilderMetadata{
			Buildpacks: buildpacks,
			Groups:     groups,
		}
	})

	it.After(func() {
		dagg.Destroy()
	})

	when("when the node_modules are vendored", func() {
		it("should build a working OCI image for a simple app", func() {
			app, err := dagg.Pack(filepath.Join(rootDir, "fixtures", "simple_app_vendored"), builderMetadata)
			Expect(err).ToNot(HaveOccurred())

			err = app.Start()
			Expect(err).ToNot(HaveOccurred())

			err = app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	when("when the node_modules are not vendored", func() {
		it("should build a working OCI image for a simple app", func() {
			app, err := dagg.Pack(filepath.Join(rootDir, "fixtures", "simple_app"), builderMetadata)
			Expect(err).ToNot(HaveOccurred())

			err = app.Start()
			Expect(err).ToNot(HaveOccurred())

			err = app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
		})
	})
}
