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

		pullPolicy              = "never"
		extenderBuildStr        = ""
		extenderBuildStrEscaped = ""
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()

		if settings.Extensions.UbiNodejsExtension.Online != "" {
			pullPolicy = "always"
			extenderBuildStr = "[extender (build)] "
			extenderBuildStrEscaped = `\[extender \(build\)\] `
		}
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
				WithPullPolicy(pullPolicy).
				WithExtensions(
					settings.Extensions.UbiNodejsExtension.Online,
				).
				WithBuildpacks(
					settings.Buildpacks.NodeEngine.Online,
					settings.Buildpacks.NPMInstall.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).NotTo(HaveOccurred(), logs.String)

			Expect(logs).To(ContainLines(
				fmt.Sprintf("%s%s 1.2.3", extenderBuildStr, settings.Buildpack.Name),
				extenderBuildStr+"  Resolving installation process",
				extenderBuildStr+"    Process inputs:",
				extenderBuildStr+"      node_modules      -> \"Not found\"",
				extenderBuildStr+"      npm-cache         -> \"Not found\"",
				extenderBuildStr+"      package-lock.json -> \"Not found\"",
				extenderBuildStr+"",
				extenderBuildStr+"    Selected NPM build process: 'npm install'",
			))
			Expect(logs).To(ContainLines(
				extenderBuildStr+"  Executing launch environment install process",
				fmt.Sprintf(extenderBuildStr+"    Running 'npm install --unsafe-perm --cache /layers/%s/npm-cache'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))
			Expect(logs).To(ContainLines(
				MatchRegexp(extenderBuildStrEscaped + `      Completed in (\d+\.\d+|\d{3})`),
			))
			Expect(logs).To(ContainLines(
				extenderBuildStr+"  Configuring launch environment",
				extenderBuildStr+"    NODE_PROJECT_PATH   -> \"/workspace\"",
				extenderBuildStr+"    NPM_CONFIG_LOGLEVEL -> \"error\"",
				fmt.Sprintf(extenderBuildStr+"    PATH                -> \"$PATH:/layers/%s/launch-modules/node_modules/.bin\"", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
			))
		})

		context("when there are build and launch modules required", func() {
			it("logs useful information for the user", func() {
				var err error
				source, err = occam.Source(filepath.Join("testdata", "dev_dependencies_during_build"))
				Expect(err).ToNot(HaveOccurred())

				var logs fmt.Stringer
				image, logs, err = pack.WithNoColor().Build.
					WithPullPolicy(pullPolicy).
					WithExtensions(
						settings.Extensions.UbiNodejsExtension.Online,
					).
					WithBuildpacks(
						settings.Buildpacks.NodeEngine.Online,
						settings.Buildpacks.NPMInstall.Online,
						settings.Buildpacks.BuildPlan.Online,
					).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred(), logs.String)

				Expect(logs).To(ContainLines(
					fmt.Sprintf("%s%s 1.2.3", extenderBuildStr, settings.Buildpack.Name),
					extenderBuildStr+"  Resolving installation process",
					extenderBuildStr+"    Process inputs:",
					extenderBuildStr+"      node_modules      -> \"Not found\"",
					extenderBuildStr+"      npm-cache         -> \"Not found\"",
					extenderBuildStr+"      package-lock.json -> \"Not found\"",
					extenderBuildStr+"",
					extenderBuildStr+"    Selected NPM build process: 'npm install'"))
				Expect(logs).To(ContainLines(
					extenderBuildStr+"  Executing build environment install process",
					fmt.Sprintf(extenderBuildStr+"    Running 'npm install --unsafe-perm --cache /layers/%s/npm-cache'", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
				))
				Expect(logs).To(ContainLines(
					MatchRegexp(extenderBuildStrEscaped + `      Completed in (\d+\.\d+|\d{3})`),
				))
				Expect(logs).To(ContainLines(
					extenderBuildStr+"  Configuring build environment",
					extenderBuildStr+"    NODE_ENV -> \"development\"",
					fmt.Sprintf(extenderBuildStr+"    PATH     -> \"$PATH:/layers/%s/build-modules/node_modules/.bin\"", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
					extenderBuildStr+"",
					fmt.Sprintf(extenderBuildStr+`  Generating SBOM for /layers/%s/build-modules`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
					MatchRegexp(extenderBuildStrEscaped+`      Completed in (\d+)(\.\d+)?(ms|s)`),
				))
				Expect(logs).To(ContainLines(
					extenderBuildStr+"  Executing launch environment install process",
					extenderBuildStr+"    Running 'npm prune'",
				))
				Expect(logs).To(ContainLines(
					extenderBuildStr+"  Configuring launch environment",
					extenderBuildStr+"    NODE_PROJECT_PATH   -> \"/workspace\"",
					extenderBuildStr+"    NPM_CONFIG_LOGLEVEL -> \"error\"",
					fmt.Sprintf(extenderBuildStr+"    PATH                -> \"$PATH:/layers/%s/launch-modules/node_modules/.bin\"", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
					extenderBuildStr+"",
					fmt.Sprintf(extenderBuildStr+`  Generating SBOM for /layers/%s/launch-modules`, strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
					MatchRegexp(extenderBuildStrEscaped+`      Completed in (\d+)(\.\d+)?(ms|s)`),
				))
			})

		})
	})
}
