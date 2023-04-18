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

func testVendored(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack().WithNoColor()
		docker = occam.NewDocker()
	})

	context("when the node_modules are vendored", func() {
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

			source, err = occam.Source(filepath.Join("testdata", "vendored"))
			Expect(err).NotTo(HaveOccurred())
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
				WithPullPolicy("never").
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
				fmt.Sprintf("%s 1.2.3", settings.Buildpack.Name),
				"  Resolving installation process",
				"    Process inputs:",
				"      node_modules      -> \"Found\"",
				"      npm-cache         -> \"Not found\"",
				"      package-lock.json -> \"Found\"",
				"",
				"    Selected NPM build process: 'npm rebuild'"))
			Expect(logs).To(ContainLines("  Executing launch environment install process"))
			Expect(logs).To(ContainLines("    Running 'npm run-script preinstall --if-present'"))
			Expect(logs).To(ContainLines(MatchRegexp(`    Running 'npm rebuild --nodedir=/layers/.+/node'`)))
			Expect(logs).To(ContainLines("    Running 'npm run-script postinstall --if-present'"))
			Expect(logs).To(ContainLines(
				"  Configuring launch environment",
				"    NODE_PROJECT_PATH   -> \"/workspace\"",
				"    NPM_CONFIG_LOGLEVEL -> \"error\"",
				fmt.Sprintf("    PATH                -> \"$PATH:/layers/%s/launch-modules/node_modules/.bin\"", strings.ReplaceAll(settings.Buildpack.ID, "/", "_")),
				"",
			))
		})

		context("when the npm and node buildpacks are cached", func() {
			it("does not reach out to the internet", func() {
				var err error
				image, _, err = pack.Build.
					WithPullPolicy("never").
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
