package integration_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testNativeModules(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker

		pullPolicy              = "never"
		extenderBuildStr        = ""
		extenderBuildStrEscaped = ""
	)

	it.Before(func() {
		pack = occam.NewPack().WithNoColor()
		docker = occam.NewDocker()
	})

	context("when there are node modules that need to be compiled", func() {
		var (
			image     occam.Image
			container occam.Container

			name   string
			source string
		)

		it.Before(func() {
			var err error
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())

			source, err = occam.Source(filepath.Join("testdata", "with_native_modules"))
			Expect(err).NotTo(HaveOccurred())

			if settings.Extensions.UbiNodejsExtension.Online != "" {
				pullPolicy = "always"
				extenderBuildStr = "[extender (build)] "
				extenderBuildStrEscaped = `\[extender \(build\)\] `
			}
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("builds a working OCI image for a simple app", func() {
			var (
				err  error
				logs fmt.Stringer
			)

			image, logs, err = pack.Build.
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
			Expect(err).NotTo(HaveOccurred())

			container, err = docker.Container.Run.
				WithCommand("npm start").
				WithEnv(map[string]string{"PORT": "8080"}).
				WithPublish("8080").
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(container).Should(BeAvailable())

			response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("8080")))
			Expect(err).NotTo(HaveOccurred())
			Expect(response.StatusCode).To(Equal(http.StatusOK))

			content, err := io.ReadAll(response.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("Hello, World!"))

			Expect(logs).To(ContainLines(
				fmt.Sprintf("%s%s 1.2.3", extenderBuildStr, settings.Buildpack.Name),
				extenderBuildStr+"  Resolving installation process",
				extenderBuildStr+"    Process inputs:",
				extenderBuildStr+"      node_modules      -> \"Found\"",
				extenderBuildStr+"      npm-cache         -> \"Not found\"",
				extenderBuildStr+"      package-lock.json -> \"Found\"",
				extenderBuildStr+"",
				extenderBuildStr+"    Selected NPM build process: 'npm rebuild'"))
			Expect(logs).To(ContainLines(extenderBuildStr + "  Executing launch environment install process"))
			Expect(logs).To(ContainLines(extenderBuildStr + "    Running 'npm run-script preinstall --if-present'"))
			if extenderBuildStrEscaped != "" {
				Expect(logs).To(ContainLines(MatchRegexp(extenderBuildStrEscaped + `    Running 'npm rebuild --nodedir='`)))
			} else {
				Expect(logs).To(ContainLines(MatchRegexp(`    Running 'npm rebuild --nodedir=/layers/.+/node'`)))
			}
			Expect(logs).To(ContainLines(extenderBuildStr + "    Running 'npm run-script postinstall --if-present'"))
			Expect(logs).To(ContainLines(
				extenderBuildStr+"  Configuring launch environment",
				extenderBuildStr+"    NODE_PROJECT_PATH   -> \"/workspace\"",
				extenderBuildStr+"    NPM_CONFIG_LOGLEVEL -> \"error\"",
				fmt.Sprintf(extenderBuildStr+"    PATH                -> \"$PATH:/layers/%s/launch-modules/node_modules/.bin\"", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
				extenderBuildStr+"",
			))
		})

		context("when the npm and node buildpacks are cached", func() {

			//UBI does not support offline installation at the moment,
			//so we are skipping it.
			if settings.Extensions.UbiNodejsExtension.Online != "" {
				return
			}

			it("does not reach out to the internet", func() {
				var err error
				image, _, err = pack.Build.
					WithPullPolicy(pullPolicy).
					WithExtensions(
						settings.Extensions.UbiNodejsExtension.Online,
					).
					WithBuildpacks(
						settings.Buildpacks.NodeEngine.Offline,
						settings.Buildpacks.NPMInstall.Online,
						settings.Buildpacks.BuildPlan.Online,
					).
					WithNetwork("none").
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred())

				container, err = docker.Container.Run.
					WithCommand("npm start").
					WithEnv(map[string]string{"PORT": "8080"}).
					WithPublish("8080").
					Execute(image.ID)
				Expect(err).NotTo(HaveOccurred())

				Eventually(container).Should(BeAvailable())

				response, err := http.Get(fmt.Sprintf("http://localhost:%s", container.HostPort("8080")))
				Expect(err).NotTo(HaveOccurred())
				Expect(response.StatusCode).To(Equal(http.StatusOK))
			})
		})
	})
}
