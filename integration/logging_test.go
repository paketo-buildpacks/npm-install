package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testLogging(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when the buildpack is run with pack build", func() {
		var (
			image occam.Image

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("logs useful information for the user", func() {
			var err error
			source, err = occam.Source(filepath.Join("testdata", "simple_app"))
			Expect(err).ToNot(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					nodeURI,
					buildpackURI,
					buildPlanURI,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			Expect(logs).To(ContainLines(
				fmt.Sprintf("%s 1.2.3", buildpackInfo.Buildpack.Name),
				"  Resolving installation process",
				"    Process inputs:",
				"      node_modules      -> \"Not found\"",
				"      npm-cache         -> \"Not found\"",
				"      package-lock.json -> \"Not found\"",
				"",
				"    Selected NPM build process: 'npm install'",
				"",
				"  Executing build process",
				fmt.Sprintf("    Running 'npm install --unsafe-perm --cache /layers/%s/npm-cache'", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
				MatchRegexp(`      Completed in (\d+\.\d+|\d{3})`),
				"",
				"  Configuring environment",
				"    NPM_CONFIG_LOGLEVEL   -> \"error\"",
				"    NPM_CONFIG_PRODUCTION -> \"true\"",
				fmt.Sprintf("    PATH                  -> \"$PATH:/layers/%s/modules/node_modules/.bin\"", strings.ReplaceAll(buildpackInfo.Buildpack.ID, "/", "_")),
				"",
			))
		})
	})
}
