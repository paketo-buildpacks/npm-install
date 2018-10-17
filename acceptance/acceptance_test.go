package acceptance

import (
	"path/filepath"

	"github.com/buildpack/libbuildpack"
	"github.com/cloudfoundry/dagger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("NPM buildpack", func() {
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

	BeforeEach(func() {
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

	AfterEach(func() {
		dagg.Destroy()
	})

	Context("when the node_modules are vendored", func() {
		It("should build a working OCI image for a simple app", func() {
			app, err := dagg.Pack(filepath.Join(rootDir, "fixtures", "simple_app_vendored"), builderMetadata)
			Expect(err).ToNot(HaveOccurred())

			err = app.Start()
			Expect(err).ToNot(HaveOccurred())

			err = app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("when the node_modules are not vendored", func() {
		It("should build a working OCI image for a simple app", func() {
			app, err := dagg.Pack(filepath.Join(rootDir, "fixtures", "simple_app"), builderMetadata)
			Expect(err).ToNot(HaveOccurred())

			err = app.Start()
			Expect(err).ToNot(HaveOccurred())

			err = app.HTTPGet("/")
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
